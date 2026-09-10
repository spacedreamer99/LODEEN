package com.lodeen.server.net;

import com.lodeen.shared.network.PlayerState;
import io.netty.channel.Channel;
import io.netty.channel.group.ChannelGroup;
import io.netty.channel.group.DefaultChannelGroup;
import io.netty.util.concurrent.GlobalEventExecutor;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

public class PlayerRegistry {
    private static final Map<Channel, PlayerState> players = new ConcurrentHashMap<>();
    private static final ChannelGroup all = new DefaultChannelGroup(GlobalEventExecutor.INSTANCE);

    public static void add(Channel ch, PlayerState state) {
        players.put(ch, state);
        all.add(ch);
    }

    public static void remove(Channel ch) {
        players.remove(ch);
        all.remove(ch);
    }

    public static void update(Channel ch, PlayerState state) {
        players.put(ch, state);
    }

    public static PlayerState get(Channel ch) { return players.get(ch); }

    public static List<PlayerState> snapshot() {
        return new ArrayList<>(players.values());
    }

    public static ChannelGroup channels() { return all; }
}
