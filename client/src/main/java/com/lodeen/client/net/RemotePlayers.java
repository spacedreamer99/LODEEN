package com.lodeen.client.net;

import com.lodeen.shared.network.PlayerState;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

public class RemotePlayers {
    private static final Map<String, PlayerInterp> interps = new ConcurrentHashMap<>();

    public static void update(List<PlayerState> states, String myId) {
        long now = System.nanoTime();
        if (states != null) {
            for (PlayerState s : states) {
                if (s.id == null) continue;
                if (myId != null && myId.equals(s.id)) continue;
                PlayerInterp existing = interps.get(s.id);
                if (existing == null) interps.put(s.id, new PlayerInterp(s, now));
                else existing.update(s, now);
            }
        }
    }

    public static List<PlayerState> sampleAll() {
        long now = System.nanoTime();
        List<PlayerState> out = new ArrayList<>(interps.size());
        for (PlayerInterp p : interps.values()) {
            PlayerState s = p.sample(now);
            if (s != null) out.add(s);
        }
        return out;
    }

    public static void clear() { interps.clear(); }
}
