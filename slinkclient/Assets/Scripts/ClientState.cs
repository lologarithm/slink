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
    public Camera mainCam;

	private NetworkMessenger net;
	private uint mySnake;

	private Queue<NetPacket> message_queue = new Queue<NetPacket>();
	private Dictionary<uint, Multipart[]> multipart_cache = new Dictionary<uint, Multipart[]>();

    // Account
    private uint accountID;

	// Game state
	private GameInstance game;
	private long latencyms;

	// Unity objects state
	private Dictionary<uint, GameObject> segments = new Dictionary<uint, GameObject> ();
	private Text latencyText;

    // Unity lifecycle methods
	void Start()
	{
        // Setup component and connect to network, send create/join messages!
		this.latencyText = latencyTextContainer.GetComponent<Text>();

        // TODO: allow handoff of network messenger from another scene?
        // Or do we pass the entire client state manager from scene to scene?
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
        if (this.updateGame()) // Only allow dir changes on ticks.
        {
            int centerX = Screen.width / 2;
            int centerY = Screen.height / 2;
            Vector2 move = new Vector2(Input.mousePosition.x - centerX, Input.mousePosition.y - centerY);
            move.Normalize();
            move.Scale(new Vector2(100,100));

            Vect2 myfacing = this.game.entities[this.mySnake].Facing;

            if (myfacing.X != (int)move.x || myfacing.Y != (int)move.y) {
                myfacing.X = (int)move.x;
                myfacing.Y = (int)move.y;
                this.SetDirection(move);
            }
        }
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

    public void SetDirection(Vector2 dir) {
        SetDirection dir_msg = new SetDirection();
        dir_msg.ID = this.mySnake;
        dir_msg.TickID = this.game.Tick;
        dir_msg.Facing = new Vect2();
        dir_msg.Facing.X = (int)(dir.x);
        dir_msg.Facing.Y = (int)dir.y;
        
        this.net.sendNetPacket(MsgType.SetDirection, dir_msg);
    }


    // Message processing
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
                if (lr.Success == 0)
                {
                    Debug.Log("Failed to login!");
                }
			    // TODO
				break;
			case MsgType.CreateAcctResp:
				CreateAcctResp car = ((CreateAcctResp)parsedMsg);
				this.accountID = car.AccountID;
				break;
            case MsgType.GameConnected:
                GameConnected gc = ((GameConnected)parsedMsg);
                this.game = new GameInstance();
                this.game.ID = gc.ID;
                this.loadEntities(gc.Entities, gc.Snakes);
                this.game.StartTime = DateTime.Now - new TimeSpan((Int64)this.latencyms * 10000);
                this.game.StartTick = gc.TickID;
                this.game.Tick = gc.TickID;
                this.mySnake = gc.SnakeID;
				break;
            case MsgType.GameMasterFrame:
                GameMasterFrame gmf = ((GameMasterFrame)parsedMsg);
                this.game.entities.Clear();
                this.loadEntities(gmf.Entities, gmf.Snakes);
                this.game.MasterTick = gmf.Tick;

				// Now update local entities based on difference between master tick and now.
                foreach (KeyValuePair<uint, PlayerSnake> entry in this.game.players)
                {
                    PlayerSnake snake = entry.Value;
                    int nticks = (int)(this.game.Tick - this.game.MasterTick);
                    snake.Move(nticks, this.game.TicksPerSecond);
                }

                Debug.Log("masterframe for gameID:" + gmf.ID.ToString());
                Debug.Log("Current tick: " + this.game.Tick + " Master Tick: " + gmf.Tick);
                break;
		}
	}

    private void loadEntities(Entity[] ents, Snake[] snakes) {
        foreach (Entity e in ents) {
            this.game.entities[e.ID] = e;
        }
        foreach (Snake s in snakes) {
            Entity head = this.game.entities[s.ID];
            PlayerSnake ps = new PlayerSnake();
            ps.ID = s.ID;
            ps.speed = s.Speed;
            ps.name = s.Name;
            ps.segments = new Entity[s.Segments.Length+1];
            ps.segments[0] = head;
            for (int j = 0; j < s.Segments.Length; j++)
            {
                ps.segments[j+1] = this.game.entities[s.Segments[j]];
            }
            this.game.players[s.ID] = ps;
        }
    }

    private bool updateGame() {
        this.game.UpdateTick();

        if (this.game.LastTickUpdated == this.game.Tick) {
            return false;
        }

        // Either a) create new snake at location or b) update existing snake based on speed.
        foreach(KeyValuePair<uint, PlayerSnake> entry in this.game.players) {
            PlayerSnake snake = entry.Value;

            int nticks = (int)(this.game.Tick - this.game.LastTickUpdated);
            snake.Move(nticks, this.game.TicksPerSecond);

            foreach (Entity e in snake.segments)
            {
                if (!this.segments.ContainsKey(e.ID))
                {
                    if (e.EType == 1) // Head
                    {
                        GameObject newseg = (GameObject)Instantiate(this.segmentPrefab, new Vector3(e.X, e.Y, 0), Quaternion.identity);
                        newseg.name = "head" + e.ID.ToString();
                        this.segments[e.ID] = newseg;
                    }
                    else if (e.EType == 2) // Body segment
                    {
                        GameObject newseg = (GameObject)Instantiate(this.segmentPrefab, new Vector3(e.X, e.Y, 0), Quaternion.identity);
                        newseg.name = "segment" + e.ID.ToString();
                        this.segments[e.ID] = newseg; 
                    }
                }
                this.updateEntityPos(e);
            }
        }
        this.updateCamera();
        this.latencyText.text = "Latency: " + this.latencyms;
        this.game.LastTickUpdated = this.game.Tick;
        return true;
    }

    private void updateEntityPos(Entity e) {
        GameObject segObj = this.segments[e.ID];
        float scale = (float)(e.Size) / (float)256.0; // Image is 256x256.
        segObj.transform.localScale = new Vector3(scale, scale, 1);
        segObj.transform.localPosition = new Vector3(e.X, e.Y, 0);
    }

    private void updateCamera() 
    {
        Entity mysnake = this.game.entities[this.mySnake];
        this.mainCam.transform.position = new Vector3(mysnake.X, mysnake.Y, -100);
    }
}

public class GameInstance
{
	public uint ID;
	public string Name;
    public Dictionary<uint, Entity> entities = new Dictionary<uint, Entity>();
    public Dictionary<uint, PlayerSnake> players = new Dictionary<uint, PlayerSnake>(); // List of players

    public float TicksPerSecond = 50; // TODO: what do
    public float TickLength = 20; // TODO: what do
        
	public uint LastTickUpdated;
	public uint Tick;

	public uint MasterTick;

	public uint StartTick;
	public DateTime StartTime;

	public void UpdateTick() {
		this.Tick = this.StartTick + (uint)( (DateTime.Now - this.StartTime).TotalMilliseconds / TickLength); // TODO: should this be hardcoded?
	}
}

public class PlayerSnake
{
	public uint ID;
	public string name;
    public int speed;
	public Entity[] segments;

    public void Move(int nticks, float tickPerSecond)
    {
        for (int i = this.segments.Length - 1; i > 0; i--)
        {
            this.segments[i].X = this.segments[i - 1].X;
            this.segments[i].Y = this.segments[i - 1].Y;
        }
        double spPerTick = ((float)this.speed) / tickPerSecond;
        this.segments[0].X += (int)((this.segments[0].Facing.X * (spPerTick*nticks))/100.0); 
        this.segments[0].Y += (int)((this.segments[0].Facing.Y * (spPerTick*nticks))/100.0);
    }
}
