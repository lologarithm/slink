package slinkserv

import (
	"math/rand"

	"github.com/lologarithm/slink/slinkserv/messages"
	"github.com/lologarithm/survival/physics"
	"github.com/lologarithm/survival/physics/quadtree"
)

type Snake struct {
	*Entity            // Snake entity itself is the head.
	Segments []*Entity // Segments are the body!
	Speed    int32     // Velocity!
	Turning  int16     // -1 Left, 0 straight, 1 = right
}

func (s *Snake) toSnakeMsg() *messages.Snake {
	segIDs := make([]uint32, len(s.Segments))
	for i, seg := range s.Segments {
		segIDs[i] = seg.ID
	}
	return &messages.Snake{
		ID:       s.ID,
		Segments: segIDs,
		Speed:    s.Speed,
		Name:     s.Name,
		Turning:  s.Turning,
	}
}

// NewSnake creates a new snake at a random location.
func NewSnake(id uint32, name string) *Snake {
	pos := physics.Vect2{
		X: int32(rand.Intn(10000) - 5000),
		Y: int32(rand.Intn(10000) - 5000),
	}
	snake := &Snake{
		Entity: &Entity{
			ID:       id,
			EType:    ETypeHead,
			Position: pos,
			Facing: physics.Vect2{
				X: 0,
				Y: 100,
			},
			Size: 300,
			Name: name,
		},
		Segments: []*Entity{},
		Speed:    2000,
	}
	for i := 0; i < 10; i++ {
		e := &Entity{
			ID:    id + uint32(i+1),
			EType: ETypeSegment,
			Position: physics.Vect2{
				X: pos.X,
				Y: pos.Y - (snake.Size/2)*int32(i+1),
			},
			Facing: snake.Entity.Facing,
			Size:   300,
		}
		snake.Segments = append(snake.Segments, e)
	}
	return snake
}

// Entity represents a single object in the game.
type Entity struct {
	ID       uint32
	Name     string
	EType    uint16
	Size     int32         // Radius
	Position physics.Vect2 // Center of entity
	Facing   physics.Vect2
}

func (e *Entity) Clone() quadtree.BoundingBoxer {
	ne := &Entity{}
	*ne = *e
	ne.Position = e.Position
	ne.Facing = e.Facing
	return ne
}

func (e *Entity) toMsg() *messages.Entity {
	o := &messages.Entity{
		ID:    e.ID,
		EType: e.EType,
		X:     e.Position.X,
		Y:     e.Position.Y,
		Size:  e.Size,
		Facing: &messages.Vect2{
			X: e.Facing.X,
			Y: e.Facing.Y,
		},
	}

	return o
}

// Intersects calculates if two entities overlap exactly.
// Used as a more fine-grained check after the quadtree check.
func (e *Entity) Intersects(o *Entity) bool {
	// TODO: (R0-R1)^2 <= (x0-x1)^2+(y0-y1)^2 <= (R0+R1)^2
	xdiff := (e.Position.X - o.Position.X)
	ydiff := (e.Position.Y - o.Position.Y)
	centerdist := xdiff*xdiff + ydiff*ydiff

	sizesum := e.Size + o.Size
	// This means either intersects OR contains
	return centerdist <= sizesum*sizesum

	// sizediff := (e.Size - o.Size)
	// return sizediff*sizediff <= centerdist // is the circle contained?
}

// Bounds is used for quadtree bounding checks.
func (e *Entity) Bounds() quadtree.BoundingBox {
	return quadtree.BoundingBox{
		MinX: e.Position.X - e.Size,
		MaxX: e.Position.X + e.Size,
		MinY: e.Position.Y - e.Size,
		MaxY: e.Position.Y + e.Size,
	}
}

// SetBounds will set this entity to a new location
func (e *Entity) SetBounds(box quadtree.BoundingBox) {
	e.Position.X = box.MinX + (box.SizeX() / 2)
	e.Position.Y = box.MinY + (box.SizeY() / 2)
}

// BoxID is used for quadtree intersection checks.
func (e *Entity) BoxID() uint32 {
	return e.ID
}
