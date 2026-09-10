package com.lodeen.shared.network;

import java.util.ArrayList;
import java.util.List;

public class WorldSnapshotPacket extends Packet {
    public List<PlayerState> players = new ArrayList<>();
    public WorldSnapshotPacket() {}
    public WorldSnapshotPacket(List<PlayerState> players) { this.players = players; }
    @Override public String id() { return "worldsnapshot"; }
}
