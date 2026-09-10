package com.lodeen.server.db;

public record PlayerRecord(
    String id,
    String name,
    long registeredAt,
    long lastSeen,
    long playtimeSec
) {
    public static PlayerRecord newPlayer(String id, String name) {
        long now = System.currentTimeMillis() / 1000L;
        return new PlayerRecord(id, name, now, now, 0);
    }
}
