package com.lodeen.server.net;

import com.lodeen.shared.network.*;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ServerHandler extends SimpleChannelInboundHandler<String> {
    private static final Logger log = LoggerFactory.getLogger(ServerHandler.class);

    private String playerName = "unknown";
    private String clientVersion = "unknown";

    @Override public void channelActive(ChannelHandlerContext ctx) {
        log.info("Client connected: {}", ctx.channel().remoteAddress());
    }

    @Override public void channelInactive(ChannelHandlerContext ctx) {
        log.info("Client disconnected: {} (player: {})", ctx.channel().remoteAddress(), playerName);
    }

    @Override protected void channelRead0(ChannelHandlerContext ctx, String msg) {
        try {
            Packet packet = PacketCodec.decode(msg);
            log.debug("Received: {} from {}", packet.id(), ctx.channel().remoteAddress());

            if (packet instanceof HandshakePacket hs) {
                this.playerName = hs.playerName;
                this.clientVersion = hs.clientVersion;
                log.info("Handshake: player={} client={}", playerName, clientVersion);
                ServerInfoPacket info = new ServerInfoPacket("0.2.0", "LODEEN dev server", 1);
                ctx.writeAndFlush(PacketCodec.encode(info) + "\n");
            } else if (packet instanceof PingPacket ping) {
                ctx.writeAndFlush(PacketCodec.encode(new PongPacket(ping.timestamp)) + "\n");
            } else {
                log.warn("Unhandled packet: {}", packet.id());
            }
        } catch (Exception e) {
            log.error("Failed: {}", e.getMessage());
        }
    }

    @Override public void exceptionCaught(ChannelHandlerContext ctx, Throwable c) {
        log.error("Channel error: {}", c.getMessage());
        ctx.close();
    }
}
