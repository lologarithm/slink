using System;
using System.IO;
using System.Text;

interface INet {
	void Serialize(BinaryWriter buffer);
	void Deserialize(BinaryReader buffer);
}

enum MsgType : ushort {Unknown=0,Ack=1,Multipart=2,Heartbeat=3,Connected=4,Disconnected=5,CreateAcct=6,CreateAcctResp=7,Login=8,LoginResp=9,JoinGame=10,GameConnected=11,GameMasterFrame=12,Entity=13,Snake=14,TurnSnake=15,RemoveEntity=16,UpdateEntity=17,SnakeDied=18,Vect2=19}

static class Messages {
// ParseNetMessage accepts input of raw bytes from a NetMessage. Parses and returns a Net message.
public static INet Parse(ushort msgType, byte[] content) {
	INet msg = null;
	MsgType mt = (MsgType)msgType;
	switch (mt)
	{
		case MsgType.Multipart:
			msg = new Multipart();
			break;
		case MsgType.Heartbeat:
			msg = new Heartbeat();
			break;
		case MsgType.Connected:
			msg = new Connected();
			break;
		case MsgType.Disconnected:
			msg = new Disconnected();
			break;
		case MsgType.CreateAcct:
			msg = new CreateAcct();
			break;
		case MsgType.CreateAcctResp:
			msg = new CreateAcctResp();
			break;
		case MsgType.Login:
			msg = new Login();
			break;
		case MsgType.LoginResp:
			msg = new LoginResp();
			break;
		case MsgType.JoinGame:
			msg = new JoinGame();
			break;
		case MsgType.GameConnected:
			msg = new GameConnected();
			break;
		case MsgType.GameMasterFrame:
			msg = new GameMasterFrame();
			break;
		case MsgType.Entity:
			msg = new Entity();
			break;
		case MsgType.Snake:
			msg = new Snake();
			break;
		case MsgType.TurnSnake:
			msg = new TurnSnake();
			break;
		case MsgType.RemoveEntity:
			msg = new RemoveEntity();
			break;
		case MsgType.UpdateEntity:
			msg = new UpdateEntity();
			break;
		case MsgType.SnakeDied:
			msg = new SnakeDied();
			break;
		case MsgType.Vect2:
			msg = new Vect2();
			break;
	}
	MemoryStream ms = new MemoryStream(content);
	msg.Deserialize(new BinaryReader(ms));
	return msg;
}
}

public class Multipart : INet {
	public ushort ID;
	public uint GroupID;
	public ushort NumParts;
	public byte[] Content;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
		buffer.Write(this.GroupID);
		buffer.Write(this.NumParts);
		buffer.Write((Int32)this.Content.Length);
		for (int v2 = 0; v2 < this.Content.Length; v2++) {
			buffer.Write(this.Content[v2]);
		}
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt16();
		this.GroupID = buffer.ReadUInt32();
		this.NumParts = buffer.ReadUInt16();
		int l3_1 = buffer.ReadInt32();
		this.Content = new byte[l3_1];
		for (int v2 = 0; v2 < l3_1; v2++) {
			this.Content[v2] = buffer.ReadByte();
		}
	}
}

public class Heartbeat : INet {
	public long Time;
	public long Latency;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.Time);
		buffer.Write(this.Latency);
	}

	public void Deserialize(BinaryReader buffer) {
		this.Time = buffer.ReadInt64();
		this.Latency = buffer.ReadInt64();
	}
}

public class Connected : INet {

	public void Serialize(BinaryWriter buffer) {
	}

	public void Deserialize(BinaryReader buffer) {
	}
}

public class Disconnected : INet {

	public void Serialize(BinaryWriter buffer) {
	}

	public void Deserialize(BinaryReader buffer) {
	}
}

public class CreateAcct : INet {
	public string Name;
	public string Password;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write((Int32)this.Name.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Name));
		buffer.Write((Int32)this.Password.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Password));
	}

	public void Deserialize(BinaryReader buffer) {
		int l0_1 = buffer.ReadInt32();
		byte[] temp0_1 = buffer.ReadBytes(l0_1);
		this.Name = System.Text.Encoding.UTF8.GetString(temp0_1);
		int l1_1 = buffer.ReadInt32();
		byte[] temp1_1 = buffer.ReadBytes(l1_1);
		this.Password = System.Text.Encoding.UTF8.GetString(temp1_1);
	}
}

public class CreateAcctResp : INet {
	public uint AccountID;
	public string Name;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.AccountID);
		buffer.Write((Int32)this.Name.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Name));
	}

	public void Deserialize(BinaryReader buffer) {
		this.AccountID = buffer.ReadUInt32();
		int l1_1 = buffer.ReadInt32();
		byte[] temp1_1 = buffer.ReadBytes(l1_1);
		this.Name = System.Text.Encoding.UTF8.GetString(temp1_1);
	}
}

public class Login : INet {
	public string Name;
	public string Password;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write((Int32)this.Name.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Name));
		buffer.Write((Int32)this.Password.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Password));
	}

	public void Deserialize(BinaryReader buffer) {
		int l0_1 = buffer.ReadInt32();
		byte[] temp0_1 = buffer.ReadBytes(l0_1);
		this.Name = System.Text.Encoding.UTF8.GetString(temp0_1);
		int l1_1 = buffer.ReadInt32();
		byte[] temp1_1 = buffer.ReadBytes(l1_1);
		this.Password = System.Text.Encoding.UTF8.GetString(temp1_1);
	}
}

public class LoginResp : INet {
	public byte Success;
	public string Name;
	public uint AccountID;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.Success);
		buffer.Write((Int32)this.Name.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Name));
		buffer.Write(this.AccountID);
	}

	public void Deserialize(BinaryReader buffer) {
		this.Success = buffer.ReadByte();
		int l1_1 = buffer.ReadInt32();
		byte[] temp1_1 = buffer.ReadBytes(l1_1);
		this.Name = System.Text.Encoding.UTF8.GetString(temp1_1);
		this.AccountID = buffer.ReadUInt32();
	}
}

public class JoinGame : INet {

	public void Serialize(BinaryWriter buffer) {
	}

	public void Deserialize(BinaryReader buffer) {
	}
}

public class GameConnected : INet {
	public uint ID;
	public uint SnakeID;
	public uint TickID;
	public Entity[] Entities;
	public Snake[] Snakes;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
		buffer.Write(this.SnakeID);
		buffer.Write(this.TickID);
		buffer.Write((Int32)this.Entities.Length);
		for (int v2 = 0; v2 < this.Entities.Length; v2++) {
			this.Entities[v2].Serialize(buffer);
		}
		buffer.Write((Int32)this.Snakes.Length);
		for (int v2 = 0; v2 < this.Snakes.Length; v2++) {
			this.Snakes[v2].Serialize(buffer);
		}
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt32();
		this.SnakeID = buffer.ReadUInt32();
		this.TickID = buffer.ReadUInt32();
		int l3_1 = buffer.ReadInt32();
		this.Entities = new Entity[l3_1];
		for (int v2 = 0; v2 < l3_1; v2++) {
			this.Entities[v2] = new Entity();
			this.Entities[v2].Deserialize(buffer);
		}
		int l4_1 = buffer.ReadInt32();
		this.Snakes = new Snake[l4_1];
		for (int v2 = 0; v2 < l4_1; v2++) {
			this.Snakes[v2] = new Snake();
			this.Snakes[v2].Deserialize(buffer);
		}
	}
}

public class GameMasterFrame : INet {
	public uint ID;
	public Entity[] Entities;
	public Snake[] Snakes;
	public uint Tick;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
		buffer.Write((Int32)this.Entities.Length);
		for (int v2 = 0; v2 < this.Entities.Length; v2++) {
			this.Entities[v2].Serialize(buffer);
		}
		buffer.Write((Int32)this.Snakes.Length);
		for (int v2 = 0; v2 < this.Snakes.Length; v2++) {
			this.Snakes[v2].Serialize(buffer);
		}
		buffer.Write(this.Tick);
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt32();
		int l1_1 = buffer.ReadInt32();
		this.Entities = new Entity[l1_1];
		for (int v2 = 0; v2 < l1_1; v2++) {
			this.Entities[v2] = new Entity();
			this.Entities[v2].Deserialize(buffer);
		}
		int l2_1 = buffer.ReadInt32();
		this.Snakes = new Snake[l2_1];
		for (int v2 = 0; v2 < l2_1; v2++) {
			this.Snakes[v2] = new Snake();
			this.Snakes[v2].Deserialize(buffer);
		}
		this.Tick = buffer.ReadUInt32();
	}
}

public class Entity : INet {
	public uint ID;
	public ushort EType;
	public int X;
	public int Y;
	public int Size;
	public Vect2 Facing;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
		buffer.Write(this.EType);
		buffer.Write(this.X);
		buffer.Write(this.Y);
		buffer.Write(this.Size);
		this.Facing.Serialize(buffer);
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt32();
		this.EType = buffer.ReadUInt16();
		this.X = buffer.ReadInt32();
		this.Y = buffer.ReadInt32();
		this.Size = buffer.ReadInt32();
		this.Facing = new Vect2();
		this.Facing.Deserialize(buffer);
	}
}

public class Snake : INet {
	public uint ID;
	public string Name;
	public uint[] Segments;
	public int Speed;
	public short Turning;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
		buffer.Write((Int32)this.Name.Length);
		buffer.Write(System.Text.Encoding.UTF8.GetBytes(this.Name));
		buffer.Write((Int32)this.Segments.Length);
		for (int v2 = 0; v2 < this.Segments.Length; v2++) {
			buffer.Write(this.Segments[v2]);
		}
		buffer.Write(this.Speed);
		buffer.Write(this.Turning);
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt32();
		int l1_1 = buffer.ReadInt32();
		byte[] temp1_1 = buffer.ReadBytes(l1_1);
		this.Name = System.Text.Encoding.UTF8.GetString(temp1_1);
		int l2_1 = buffer.ReadInt32();
		this.Segments = new uint[l2_1];
		for (int v2 = 0; v2 < l2_1; v2++) {
			this.Segments[v2] = buffer.ReadUInt32();
		}
		this.Speed = buffer.ReadInt32();
		this.Turning = buffer.ReadInt16();
	}
}

public class TurnSnake : INet {
	public uint ID;
	public short Direction;
	public uint TickID;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
		buffer.Write(this.Direction);
		buffer.Write(this.TickID);
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt32();
		this.Direction = buffer.ReadInt16();
		this.TickID = buffer.ReadUInt32();
	}
}

public class RemoveEntity : INet {
	public Entity Ent;

	public void Serialize(BinaryWriter buffer) {
		this.Ent.Serialize(buffer);
	}

	public void Deserialize(BinaryReader buffer) {
		this.Ent = new Entity();
		this.Ent.Deserialize(buffer);
	}
}

public class UpdateEntity : INet {
	public Entity Ent;

	public void Serialize(BinaryWriter buffer) {
		this.Ent.Serialize(buffer);
	}

	public void Deserialize(BinaryReader buffer) {
		this.Ent = new Entity();
		this.Ent.Deserialize(buffer);
	}
}

public class SnakeDied : INet {
	public uint ID;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.ID);
	}

	public void Deserialize(BinaryReader buffer) {
		this.ID = buffer.ReadUInt32();
	}
}

public class Vect2 : INet {
	public int X;
	public int Y;

	public void Serialize(BinaryWriter buffer) {
		buffer.Write(this.X);
		buffer.Write(this.Y);
	}

	public void Deserialize(BinaryReader buffer) {
		this.X = buffer.ReadInt32();
		this.Y = buffer.ReadInt32();
	}
}

