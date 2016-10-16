using UnityEngine;
using System;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Collections.Generic;

internal class NetworkMessenger
{
	Socket sending_socket = new Socket(AddressFamily.InterNetwork, SocketType.Dgram, ProtocolType.Udp);
	IPAddress send_to_address;
	IPEndPoint sending_end_point;

	// Caching network state
	private byte[] buff = new byte[8192];
	private byte[] stored_bytes = new byte[8192];
	private int numStored = 0;

	private uint multi_groupid = 0;

	private Queue<NetPacket> message_queue = new Queue<NetPacket>();

	public NetworkMessenger(Queue<NetPacket> queue,string addr, int port)
	{
		Debug.Log("Starting network now!");
		this.message_queue = queue;
		this.send_to_address = IPAddress.Parse(addr);
		this.sending_end_point = new IPEndPoint(send_to_address, port);
		sending_socket.Connect(this.sending_end_point);

		// Start Receive and a new Accept
		try
		{
			sending_socket.BeginReceive(this.buff, 0, this.buff.Length, SocketFlags.None, new AsyncCallback(ReceiveCallback), null);
		}
		catch (SocketException e)
		{
			// DO something
			System.Console.WriteLine(e.ToString());
		}
	}

	public void sendNetPacket(MsgType t, INet outmsg)
	{
		NetPacket msg = new NetPacket();
		MemoryStream stream = new MemoryStream();
		BinaryWriter buffer = new BinaryWriter(stream);
		outmsg.Serialize(buffer);

		if (buffer.BaseStream.Length + NetPacket.DEFAULT_FRAME_LEN > 512)
		{
			msg.message_type = (byte)MsgType.Multipart;
			//  calculate how many parts we have to split this into
			int maxsize = 512 - (12+NetPacket.DEFAULT_FRAME_LEN);
			int parts = ((int)buffer.BaseStream.Length / maxsize) + 1;
			this.multi_groupid++;
			int bstart = 0;
			for (int i = 0; i < parts; i++) {
				int bend = bstart + maxsize;
				if (i+1 == parts) {
					bend = bstart + (((int)buffer.BaseStream.Length) % maxsize);
				}
				Multipart wrapper = new Multipart();
				wrapper.ID = (ushort)i;
				wrapper.GroupID = this.multi_groupid;
				wrapper.NumParts = (ushort)parts;
				wrapper.Content = new byte[bend-bstart];
				buffer.BaseStream.Read(wrapper.Content, bstart, bend - bstart);

				MemoryStream pstream = new MemoryStream();
				BinaryWriter pbuffer = new BinaryWriter(pstream);
				wrapper.Serialize(pbuffer);

				msg.content = pstream.ToArray();
				msg.content_length = (ushort)pstream.Length;
				this.sending_socket.Send(msg.MessageBytes());
                bstart = bend;
			}
		}
		else
		{
			msg.content = stream.ToArray();
			msg.content_length = (ushort)msg.content.Length;
			msg.message_type = (byte)t;
			this.sending_socket.Send(msg.MessageBytes());
		}
	}

	private void ReceiveCallback(IAsyncResult result)
	{
		int bytesRead = 0;
		if (!sending_socket.Connected) {
			Debug.Log ("Socket disconnected stopping pocessing.");
			return;
		}
		try
		{
			bytesRead = sending_socket.EndReceive(result);
		}
		catch (SocketException exc)
		{
			CloseConnection();
			Debug.Log("Socket exception: " + exc.SocketErrorCode);
		}
		catch (Exception exc)
		{
			CloseConnection();
			Debug.Log("Exception: " + exc);
		}

		if (bytesRead > 0)
		{
			if (this.stored_bytes.Length < this.numStored + bytesRead)
			{
				byte[] newbuf = new byte[this.stored_bytes.Length * 2];
				Array.Copy(this.stored_bytes, 0, newbuf, 0, this.numStored);
				this.stored_bytes = newbuf;
			}
			Array.Copy(this.buff, 0, this.stored_bytes, this.numStored, bytesRead);
			this.numStored += bytesRead;
			ProcessBytes();
			sending_socket.BeginReceive(this.buff, 0, buff.Length, SocketFlags.None, new AsyncCallback(ReceiveCallback), null);
		}
		else
			CloseConnection();
	}

	private void ProcessBytes()
	{
		byte[] input_bytes = new byte[this.numStored];
		Array.Copy(this.stored_bytes, 0, input_bytes, 0, this.numStored);
		NetPacket nMsg = NetPacket.fromBytes(input_bytes);
		if (nMsg != null)
		{
			// Check for full content available. If so, time to add this to the processing queue.
			if (nMsg.full_content != null)
			{
				this.numStored -= nMsg.full_content.Length;
				if (nMsg==null) {
					Debug.Log("Enqueing a null message!?");
				} 
				this.message_queue.Enqueue(nMsg);
				// If we have enough bytes to start a new message we call ProcessBytes again.
				if (input_bytes.Length - nMsg.full_content.Length > NetPacket.DEFAULT_FRAME_LEN)
				{
					ProcessBytes();
				}
			}
		}
	}

	public void CloseConnection()
	{
		if (sending_socket.Connected)
		{
			// sending_socket.Send (new byte[] { 255, 0, 0, 0, 0, 0, 0 }); // TODO: create a disconnect message.
			sending_socket.Close();
		}
	}
}

public class NetPacket
{
	public const int DEFAULT_FRAME_LEN = 6;

	public ushort message_type;
	public int from_player;
	public ushort content_length;
	public ushort sequence;
	public byte[] content;
	public byte[] full_content;


	public byte[] MessageBytes()
	{
		///byte[] byte_array = new byte[]
		MemoryStream stream = new MemoryStream();
		using (BinaryWriter writer = new BinaryWriter(stream))
		{
			writer.Write(this.message_type);
			writer.Write(sequence);
			writer.Write(content_length);
			writer.Write(content);
		}
		return stream.ToArray();
	}

	public byte[] Content()
	{
		byte[] content = new byte[this.content_length];
		Array.Copy(this.full_content, DEFAULT_FRAME_LEN, content, 0, this.content_length);
		return content;
	}

	public static NetPacket fromBytes(byte[] bytes)
	{
		NetPacket newMsg = null;
		if (bytes.Length >= DEFAULT_FRAME_LEN)
		{
			newMsg = new NetPacket();
			newMsg.message_type = BitConverter.ToUInt16(bytes, 0);
			newMsg.sequence = BitConverter.ToUInt16(bytes, 2);
			newMsg.content_length = BitConverter.ToUInt16(bytes, 4);

			int totalLen = DEFAULT_FRAME_LEN + newMsg.content_length;
			if (bytes.Length >= totalLen)
			{
				newMsg.full_content = new byte[totalLen];
				Array.Copy(bytes, 0, newMsg.full_content, 0, totalLen);
			}
		}
		return newMsg;
	}

	public bool loadContent(byte[] bytes)
	{
		if (bytes.Length >= this.content_length + DEFAULT_FRAME_LEN)
		{
			Array.Copy(bytes, 0, this.full_content, 0, DEFAULT_FRAME_LEN + this.content_length);
			return true;
		}

		return false;
	}
}
