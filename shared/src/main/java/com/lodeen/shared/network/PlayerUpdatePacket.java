package com.lodeen.shared.network;

public class PlayerUpdatePacket extends Packet {
    public PlayerState state;
    public PlayerUpdatePacket() {}
    public PlayerUpdatePacket(PlayerState s) { this.state = s; }
    @Override public String id() { return "playerupdate"; }
}
