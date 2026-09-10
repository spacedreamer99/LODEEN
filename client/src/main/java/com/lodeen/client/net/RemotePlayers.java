package com.lodeen.client.net;

import com.lodeen.shared.network.PlayerState;
import java.util.List;
import java.util.concurrent.CopyOnWriteArrayList;

public class RemotePlayers {
    private static final List<PlayerState> states = new CopyOnWriteArrayList<>();

    public static void update(List<PlayerState> newStates, String myId) {
        states.clear();
        if (newStates == null) return;
        for (PlayerState s : newStates) {
            if (myId != null && myId.equals(s.id)) continue;
            states.add(s);
        }
    }

    public static List<PlayerState> all() { return states; }
}
