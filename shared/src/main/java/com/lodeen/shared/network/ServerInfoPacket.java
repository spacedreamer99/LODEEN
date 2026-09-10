package com.lodeen.shared.network;

public class ServerInfoPacket extends Packet {
    public String serverVersion;
    public String motd;
    public int playerCount;

    public ServerInfoPacket() {}
    public ServerInfoPacket(String v, String m, int pc) {
        this.serverVersion = v; this.motd = m; this.playerCount = pc;
    }
    @Override public String id() { return "serverinfo"; }
}
