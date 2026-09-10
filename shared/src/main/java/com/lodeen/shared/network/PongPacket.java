package com.lodeen.shared.network;

public class PongPacket extends Packet {
    public long timestamp;
    public PongPacket() {}
    public PongPacket(long timestamp) { this.timestamp = timestamp; }
    @Override public String id() { return "pong"; }
}
