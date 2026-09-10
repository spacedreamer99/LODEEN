package com.lodeen.client.net;

import com.lodeen.shared.network.PlayerState;
import java.util.ArrayDeque;
import java.util.Deque;

public class PlayerInterp {
    private static final long INTERP_DELAY_NS = 100_000_000L;  // 100 мс
    private static final int MAX_SNAPSHOTS = 8;

    private record Snap(PlayerState state, long time) {}
    private final Deque<Snap> buf = new ArrayDeque<>();

    public PlayerInterp(PlayerState s, long now) {
        buf.addLast(new Snap(copy(s), now));
    }

    public void update(PlayerState s, long now) {
        buf.addLast(new Snap(copy(s), now));
        while (buf.size() > MAX_SNAPSHOTS) buf.removeFirst();
    }

    public PlayerState sample(long now) {
        if (buf.isEmpty()) return null;
        if (buf.size() == 1) return buf.peekFirst().state;

        long renderAt = now - INTERP_DELAY_NS;

        Snap a = null, b = null;
        for (Snap s : buf) {
            if (s.time <= renderAt) a = s;
            if (s.time >= renderAt && b == null) b = s;
        }
        if (a == null) a = buf.peekFirst();
        if (b == null) b = buf.peekLast();
        if (a == b) return a.state;

        float t = (renderAt - a.time) / (float) (b.time - a.time);
        t = Math.max(0f, Math.min(1f, t));
        return lerpState(a.state, b.state, t);
    }

    private static PlayerState lerpState(PlayerState A, PlayerState B, float t) {
        PlayerState o = new PlayerState();
        o.id = B.id; o.name = B.name;
        o.x = A.x + (B.x - A.x) * t;
        o.y = A.y + (B.y - A.y) * t;
        o.z = A.z + (B.z - A.z) * t;
        o.yaw = lerpAngle(A.yaw, B.yaw, t);
        o.pitch = A.pitch + (B.pitch - A.pitch) * t;
        return o;
    }

    private static float lerpAngle(float a, float b, float t) {
        float d = b - a;
        while (d > 180) d -= 360;
        while (d < -180) d += 360;
        return a + d * t;
    }

    private static PlayerState copy(PlayerState s) {
        PlayerState c = new PlayerState();
        c.id = s.id; c.name = s.name;
        c.x = s.x; c.y = s.y; c.z = s.z;
        c.yaw = s.yaw; c.pitch = s.pitch;
        return c;
    }
}
