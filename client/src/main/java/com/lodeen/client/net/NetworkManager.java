package com.lodeen.client.net;

import com.lodeen.shared.network.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class NetworkManager {
    private static final Logger log = LoggerFactory.getLogger(NetworkManager.class);

    private NetworkClient client;
    private volatile boolean connected = false;
    private volatile long pingMs = -1;
    private volatile String motd = "-";
    private volatile String myId = null;
    private long lastUpdateSent = 0;

    public void connect(String host, int port, String playerName) {
        client = new NetworkClient(host, port, new NetworkClient.Listener() {
            @Override public void onConnected()    { connected = true;  log.info("Net: connected"); }
            @Override public void onDisconnected() { connected = false; log.info("Net: disconnected"); }
            @Override public void onPacket(Packet p) {
                if (p instanceof ServerInfoPacket info) {
                    motd = info.motd;
                    myId = info.yourId;
                    log.info("My player id: {}", myId);
                } else if (p instanceof WorldSnapshotPacket snap) {
                    RemotePlayers.update(snap.players, myId);
                }
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

    public void sendPlayerState(float x, float y, float z, float yaw, float pitch) {
        if (!connected) return;
        long now = System.currentTimeMillis();
        if (now - lastUpdateSent < 50) return;
        lastUpdateSent = now;
        PlayerState st = new PlayerState(null, null, x, y, z, yaw, pitch);
        client.send(new PlayerUpdatePacket(st));
    }

    public void disconnect() { if (client != null) client.disconnect(); }

    public boolean isConnected() { return connected; }
    public long getPingMs()      { return pingMs; }
    public String getMotd()      { return motd; }
}
