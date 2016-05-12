package slinkserv

import (
	"github.com/lologarithm/slink/slinkserv/messages"
	"github.com/lologarithm/survival/physics"
	"github.com/lologarithm/survival/physics/quadtree"
)

// GameWorld represents all the data in the world.
// Physical entities and the physics simulation.
type GameWorld struct {
	Entities       map[uint32]*Entity
	Snakes         map[uint32]*Snake
	Tree           quadtree.QuadTree
	TickID         uint32
	TicksPerSecond float64
	TickLength     float64
}

func NewWorld() *GameWorld {
	ticks := 50.0
	tickLength := 1000.0 / ticks
	return &GameWorld{
		Entities:       map[uint32]*Entity{},
		Snakes:         map[uint32]*Snake{},
		TicksPerSecond: ticks,
		TickLength:     tickLength,
		Tree: quadtree.NewQuadTree(quadtree.BoundingBox{
			MinX: -1000000,
			MaxX: 1000000,
			MinY: -1000000,
			MaxY: 1000000,
		}),
	}
}

// Clone returns a deep copy of the game world at this time.
func (gw *GameWorld) Clone() *GameWorld {
	nw := NewWorld()
	nw.TickID = gw.TickID
	for k, e := range gw.Entities {
		ne := &Entity{}
		*ne = *e
		ne.Position = e.Position
		ne.Facing = e.Facing

		nw.Entities[k] = ne
		nw.Tree.Add(ne)
	}

	for k, s := range gw.Snakes {
		ns := &Snake{}
		*ns = *s
		ns.Entity = nw.Entities[k]
		ns.Segments = make([]*Entity, len(s.Segments))
		for idx, seg := range s.Segments {
			ns.Segments[idx] = nw.Entities[seg.ID]
		}
		nw.Snakes[k] = ns
	}
	return nw
}

// EntitiesMsg converts all entities in the world to a network message.
func (gw *GameWorld) EntitiesMsg() []*messages.Entity {
	es := make([]*messages.Entity, len(gw.Entities))
	idx := 0
	for _, e := range gw.Entities {
		es[idx] = e.toMsg()
		idx++
	}

	return es
}

// SnakesMsg converts all snakes in the world to a network message.
func (gw *GameWorld) SnakesMsg() []*messages.Snake {
	es := make([]*messages.Snake, len(gw.Snakes))
	idx := 0
	for _, snake := range gw.Snakes {
		es[idx] = snake.toSnakeMsg()
		idx++
	}
	return es
}

func (gw *GameWorld) Tick() []Collision {
	for _, snake := range gw.Snakes {
		// Apply turning
		if snake.Turning != 0 {
			turn := -0.06
			if snake.Turning == -1 {
				turn = 0.06
			}
			snake.Facing = physics.NormalizeVect2(physics.RotateVect2(snake.Facing, turn), 100)
		}

		// Advance snake
		snakeDist := snake.Size / 3
		tickmv := (float64(snake.Speed) / gw.TicksPerSecond) / 100.0 // Dir vector normalizes to 100, so divide speed by 100
		newpos := physics.Vect2{
			X: snake.Position.X + int32(float64(snake.Facing.X)*tickmv),
			Y: snake.Position.Y + int32(float64(snake.Facing.Y)*tickmv),
		}
		snake.Position = newpos

		for _, seg := range snake.Segments {
			movevect := physics.SubVect2(newpos, seg.Position)
			mag := int32(movevect.Magnitude())
			if mag > snakeDist {
				movevect = physics.NormalizeVect2(movevect, mag-snakeDist)
				newpos = physics.Vect2{
					X: seg.Position.X + movevect.X,
					Y: seg.Position.Y + movevect.Y,
				}
				seg.Position = newpos
			}
			seg.Facing = movevect
			newpos = seg.Position
		}
	}

	// TODO: calculate collisions!
	// 1. check quad tree for possible bounding box collisions
	// 2. actually calculate exact distance and see if circles collide.
	// The only collisions that matter are snake heads!

	// If head touches body, head dies
	// If head touches head, bigger head wins.
	gw.TickID++
	return nil
}
