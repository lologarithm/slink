package slinkserv

import (
	"log"

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
	CurrentTickID  uint32 // Can be rewound to recalculate old ticks.
	RealTickID     uint32 // The actual current tick.
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
	nw.CurrentTickID = gw.CurrentTickID
	nw.RealTickID = gw.RealTickID // When we rewind old states should have the 'current' state.
	nw.Tree = *gw.Tree.Clone()
	nw.Entities = make(map[uint32]*Entity, len(gw.Entities))
	nw.Snakes = make(map[uint32]*Snake, len(gw.Snakes))

	children := gw.Tree.Query(quadtree.BoundingBox{
		MinX: -1000000,
		MaxX: 1000000,
		MinY: -1000000,
		MaxY: 1000000,
	})

	if len(children) != len(gw.Entities) {
		log.Printf("children: %d, entities: %d", len(children), len(gw.Entities))
		panic("Incorrect number of children")
	}

	for k, e := range children {
		nw.Entities[uint32(k)] = e.(*Entity)
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
		if newpos.X > 999999 {
			newpos.X = -999999
		} else if newpos.X < -999999 {
			newpos.X = 999999
		}
		if newpos.Y > 999999 {
			newpos.Y = -999999
		} else if newpos.Y < -999999 {
			newpos.Y = 999999
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
	gw.CurrentTickID++
	return nil
}
