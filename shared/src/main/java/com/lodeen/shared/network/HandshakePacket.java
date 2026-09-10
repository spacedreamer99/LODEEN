package com.lodeen.shared.network;

public class HandshakePacket extends Packet {
    public String clientVersion;
    public String playerName;
    public HandshakePacket() {}
    public HandshakePacket(String clientVersion, String playerName) {
        this.clientVersion = clientVersion;
        this.playerName = playerName;
    }
    @Override public String id() { return "handshake"; }
}
