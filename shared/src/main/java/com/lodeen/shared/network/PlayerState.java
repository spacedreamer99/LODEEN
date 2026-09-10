package com.lodeen.shared.network;

public class PlayerState {
    public String id;      // уникальный ID на сервере
    public String name;
    public float x, y, z;
    public float yaw, pitch;

    public PlayerState() {}
    public PlayerState(String id, String name, float x, float y, float z, float yaw, float pitch) {
        this.id = id; this.name = name;
        this.x = x; this.y = y; this.z = z;
        this.yaw = yaw; this.pitch = pitch;
    }
}
