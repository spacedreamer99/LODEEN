package com.lodeen.client.net;

import com.lodeen.shared.network.NetworkClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class NetworkManager {
    private static final Logger log = LoggerFactory.getLogger(NetworkManager.class);

    private NetworkClient client;
    private volatile boolean connected = false;
    private volatile long pingMs = -1;

    public void connect(String host, int port) {
        client = new NetworkClient(host, port, new NetworkClient.Listener() {
            @Override public void onConnected()    { connected = true;  log.info("Net: connected"); }
            @Override public void onDisconnected() { connected = false; log.info("Net: disconnected"); }
            @Override public void onPacket(com.lodeen.shared.network.Packet p) { /* TODO */ }
        });
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
}
