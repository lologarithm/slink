using UnityEngine;
using System;
using System.IO;
using System.Net;
using System.Collections.Generic;
using UnityEngine.UI;

public class ClientState : MonoBehaviour
{
	public GameObject segmentPrefab;
	public string serverAddress;
	public int serverPort;
	public GameObject latencyTextContainer;

	private NetworkMessenger net;
	private uint account;

	private Queue<NetPacket> message_queue = new Queue<NetPacket>();
	private Dictionary<uint, Multipart[]> multipart_cache = new Dictionary<uint, Multipart[]>();

	// Game state
	private GameInstance game;
	private long latencyms;

	// Unity objects state
	private DateTime lastUpdate;
	private Dictionary<uint, GameObject> segments = new Dictionary<uint, GameObject> ();
	private Text latencyText;

	void Start()
	{
		this.latencyText = latencyTextContainer.GetComponent<Text>();
		net = new NetworkMessenger(this.message_queue, serverAddress, serverPort);
		this.CreateAccount("asdf", "asdf");
		this.JoinGame();
	}


	// Update is called once per frame?
	void Update()
	{
		int loops = this.message_queue.Count;
		for (int i = 0; i < loops; i++)
		{
			NetPacket msg = this.message_queue.Dequeue();
			this.ParseAndProcess(msg);
		}
		// If game is null, we have nothing to do but process network.
		if (this.game == null) 
		{
			return;
		}

		this.updateGame();
	}

	private void updateGame() {
		this.game.UpdateTick();

		if (this.game.LastTickUpdated == this.game.Tick) {
			return;
		}

		for (int i = 0; i < this.game.entities.Length; i++) {
			Entity e = this.game.entities[i];

			if (this.segments.ContainsKey(e.ID)) 
			{
				if (e.EType == 1) {
					double nticks = this.game.Tick - this.game.LastTickUpdated;
					// TODO: add speed into this (right now speed is hardcoded to 100 units over 50 updates/sec)
					double spPerTick = 100.0 / 50;
					e.X += (int)(e.Facing.X * (spPerTick*nticks)); 
					e.Y += (int)(e.Facing.Y * (spPerTick*nticks));
				}

				GameObject seg = this.segments[e.ID];
				float scale = (float)(e.Size) / (float)256.0;
				seg.transform.localScale = new Vector3 (scale, scale, 1);
				seg.transform.localPosition = new Vector3 (e.X, e.Y, 0);
				continue;
			}

			if (e.EType == 1) // Head
			{
				double nticks = this.game.Tick - this.game.MasterTick;
				e.X += (int)(e.Facing.X * (100.0 / (nticks*50.0)));
				e.Y += (int)(e.Facing.Y * (100.0 / (nticks*50.0)));
				GameObject seg = (GameObject)Instantiate(this.segmentPrefab, new Vector3(e.X, e.Y, 0), Quaternion.identity);
				float scale = (float)(e.Size) / (float)256.0;
				seg.transform.localScale = new Vector3 (scale, scale, 1);
				seg.name = "head" + e.ID.ToString();
				this.segments[e.ID] = seg;
			}
			else if (e.EType == 2) // Body segment
			{
				GameObject seg = (GameObject)Instantiate(this.segmentPrefab, new Vector3(e.X, e.Y, 0), Quaternion.identity);
				float scale = (float)(e.Size) / (float)256.0;
				seg.transform.localScale = new Vector3 (scale, scale, 1);
				seg.name = "segment" + e.ID.ToString();
				this.segments[e.ID] = seg; 
			}

		}
		this.latencyText.text = "Latency: " + this.latencyms;
		this.lastUpdate = DateTime.Now;
		this.game.LastTickUpdated = this.game.Tick;
	}

	void OnApplicationQuit()
	{
		this.net.CloseConnection();
	}

	// Awake dont destroy keeps this object in memory even when we load a different scene.
	void Awake()
	{
		DontDestroyOnLoad(gameObject);
	}

	// Public functions game can call.
	public void CreateAccount(string name, string password)
	{
		CreateAcct outmsg = new CreateAcct();
		outmsg.Name = name;
		outmsg.Password = password;
		this.net.sendNetPacket(MsgType.CreateAcct, outmsg);
	}

	public void JoinGame()
	{
		JoinGame gomsg = new JoinGame();
		this.net.sendNetPacket(MsgType.JoinGame, gomsg);
	}

	public void Login(string name, string password)
	{
		Login login_msg = new Login();
		login_msg.Name = name;
		login_msg.Password = password;
		this.net.sendNetPacket(MsgType.Login, login_msg);
	}

	private void ParseAndProcess(NetPacket np)
	{
		INet parsedMsg = Messages.Parse(np.message_type, np.Content());

		// Read from message queue and process!
		// Send updates to each object.
		switch ((MsgType)np.message_type)
		{
			case MsgType.Multipart:
				Multipart mpmsg = (Multipart)parsedMsg;
				// 1. If this group doesn't exist, create it
				if (!this.multipart_cache.ContainsKey(mpmsg.GroupID))
				{
					this.multipart_cache[mpmsg.GroupID] = new Multipart[mpmsg.NumParts];
				}
				// 2. Insert message into group
				this.multipart_cache[mpmsg.GroupID][mpmsg.ID] = mpmsg;
				// 3. Check if all messages exist
				bool complete = true;
				int totalContent = 0;
				foreach (Multipart m in this.multipart_cache[mpmsg.GroupID])
				{
					if (m == null)
					{
						complete = false;
						break;
					}
					totalContent += m.Content.Length;
				}
				// 4. if so, group up bytes and call 'messages.parse' on the content
				if (complete)
				{
					byte[] content = new byte[totalContent];
					int co = 0;
					foreach (Multipart m in this.multipart_cache[mpmsg.GroupID])
					{
						System.Buffer.BlockCopy(m.Content, 0, content, co, m.Content.Length);
						co += m.Content.Length;
					}
					NetPacket newpacket = NetPacket.fromBytes(content);
					if (newpacket == null)
					{
						Debug.LogError("Multipart message content parsing failed... we done goofed");
					}
					this.ParseAndProcess(newpacket);
				}
				// 5. clean up!
				break;
			case MsgType.Heartbeat:
				Heartbeat hb = ((Heartbeat)parsedMsg);
				this.latencyms = hb.Latency;
				this.net.sendNetPacket(MsgType.Heartbeat, parsedMsg);
				break;
			case MsgType.LoginResp:
				LoginResp lr = ((LoginResp)parsedMsg);
			// TODO
				break;
			case MsgType.CreateAcctResp:
				CreateAcctResp car = ((CreateAcctResp)parsedMsg);
				this.account = car.AccountID;
				break;
			case MsgType.GameConnected:
				GameConnected gc = ((GameConnected)parsedMsg);
				this.game = new GameInstance ();
				this.game.ID = gc.ID;
				this.game.entities = gc.Entities;
				this.game.StartTime = DateTime.Now - new TimeSpan((Int64)this.latencyms*10000);
				this.game.Tick = gc.TickID;
				break;
			case MsgType.GameMasterFrame:
				GameMasterFrame gmf = ((GameMasterFrame)parsedMsg);
				Debug.Log ("masterframe for gameID:" + gmf.ID.ToString ());
				Debug.Log ("Num entities" + gmf.Entities.Length);
				this.game.entities = gmf.Entities;
				Debug.Log ("Current tick: " + this.game.Tick + " Master Tick: " + gmf.Tick);
				this.game.MasterTick = gmf.Tick;

				// Now update local entities based on difference between master tick and now.
				for (int i = 0; i < this.game.entities.Length; i++) {
					Entity e = this.game.entities [i];
					double nticks = this.game.Tick - this.game.MasterTick;
					e.X += (int)(e.Facing.X * (100.0 / (nticks * 50.0)));
					e.Y += (int)(e.Facing.Y * (100.0 / (nticks * 50.0)));
				}
                break;
		}
	}
}

public class GameInstance
{
	public uint ID;
	public string Name;
	public Entity[] entities;
	public Player[] players; // List of players

	public uint LastTickUpdated;
	public uint Tick;

	public uint MasterTick;


	public uint StartTick;
	public DateTime StartTime;

	public void UpdateTick() {
		this.Tick = this.StartTick + (uint)( (DateTime.Now - this.StartTime).TotalMilliseconds / 20.0); // TODO: should this be hardcoded?
	}
}

public class Player
{
	public uint ID;
	public string name;
	public Entity[] segments;
}
