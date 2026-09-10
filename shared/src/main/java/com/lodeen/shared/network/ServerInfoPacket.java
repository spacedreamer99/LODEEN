package com.lodeen.shared.network;

public class ServerInfoPacket extends Packet {
    public String serverVersion;
    public String motd;
    public int playerCount;
    public String yourId;

    public ServerInfoPacket() {}
    public ServerInfoPacket(String v, String m, int pc, String id) {
        this.serverVersion = v; this.motd = m; this.playerCount = pc; this.yourId = id;
    }
    @Override public String id() { return "serverinfo"; }
}
