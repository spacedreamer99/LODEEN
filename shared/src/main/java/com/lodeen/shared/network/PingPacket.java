package com.lodeen.shared.network;

public class PingPacket extends Packet {
    public long timestamp;
    public PingPacket() {}
    public PingPacket(long timestamp) { this.timestamp = timestamp; }
    @Override public String id() { return "ping"; }
}
