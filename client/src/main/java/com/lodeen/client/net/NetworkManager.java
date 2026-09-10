package com.lodeen.client.net;

import com.lodeen.shared.network.NetworkClient;
import com.lodeen.shared.network.Packet;
import com.lodeen.shared.network.ServerInfoPacket;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class NetworkManager {
    private static final Logger log = LoggerFactory.getLogger(NetworkManager.class);

    private NetworkClient client;
    private volatile boolean connected = false;
    private volatile long pingMs = -1;
    private volatile String motd = "-";

    public void connect(String host, int port, String playerName) {
        client = new NetworkClient(host, port, new NetworkClient.Listener() {
            @Override public void onConnected()    { connected = true;  log.info("Net: connected"); }
            @Override public void onDisconnected() { connected = false; log.info("Net: disconnected"); }
            @Override public void onPacket(Packet p) {
                if (p instanceof ServerInfoPacket info) motd = info.motd;
            }
        });
        client.setPlayerName(playerName);
        client.connect();
    }

    public void tick() {
        if (client == null) return;
        client.tickPing();
        pingMs = client.getLastPingMs();
    }

    public void disconnect() { if (client != null) client.disconnect(); }

    public boolean isConnected() { return connected; }
    public long getPingMs()      { return pingMs; }
    public String getMotd()      { return motd; }
}
