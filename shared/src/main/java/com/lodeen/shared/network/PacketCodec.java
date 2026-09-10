package com.lodeen.shared.network;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;

import java.util.HashMap;
import java.util.Map;

public class PacketCodec {
    private static final ObjectMapper MAPPER = new ObjectMapper();
    private static final Map<String, Class<? extends Packet>> REGISTRY = new HashMap<>();

    static {
        register("ping",      PingPacket.class);
        register("pong",      PongPacket.class);
        register("handshake", HandshakePacket.class);
        register("serverinfo", ServerInfoPacket.class);
        register("playerupdate", PlayerUpdatePacket.class);
        register("worldsnapshot", WorldSnapshotPacket.class);
    }

    public static void register(String id, Class<? extends Packet> clazz) {
        REGISTRY.put(id, clazz);
    }

    public static String encode(Packet packet) throws Exception {
        ObjectNode node = MAPPER.createObjectNode();
        node.put("_id", packet.id());
        node.set("data", MAPPER.valueToTree(packet));
        return MAPPER.writeValueAsString(node);
    }

    public static Packet decode(String json) throws Exception {
        JsonNode root = MAPPER.readTree(json);
        String id = root.get("_id").asText();
        Class<? extends Packet> clazz = REGISTRY.get(id);
        if (clazz == null) throw new IllegalArgumentException("Unknown packet: " + id);
        return MAPPER.treeToValue(root.get("data"), clazz);
    }
}
