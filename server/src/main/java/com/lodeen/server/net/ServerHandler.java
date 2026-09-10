package com.lodeen.server.net;

import com.lodeen.shared.network.*;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ServerHandler extends SimpleChannelInboundHandler<String> {
    private static final Logger log = LoggerFactory.getLogger(ServerHandler.class);

    private String playerName = "unknown";
    private String playerId;

    @Override public void channelActive(ChannelHandlerContext ctx) {
        log.info("Client connected: {}", ctx.channel().remoteAddress());
    }

    @Override public void channelInactive(ChannelHandlerContext ctx) {
        PlayerRegistry.remove(ctx.channel());
        log.info("Client disconnected: {} (player: {})", ctx.channel().remoteAddress(), playerName);
    }

    @Override protected void channelRead0(ChannelHandlerContext ctx, String msg) {
        try {
            Packet packet = PacketCodec.decode(msg);

            if (packet instanceof HandshakePacket hs) {
                this.playerName = hs.playerName;
                this.playerId = ctx.channel().id().asShortText();
                PlayerState st = new PlayerState(playerId, playerName, 0, 0, 3, 0, 0);
                PlayerRegistry.add(ctx.channel(), st);
                log.info("Handshake: player={} id={}", playerName, playerId);
                ServerInfoPacket info = new ServerInfoPacket("0.2.0", "LODEEN dev server",
                        PlayerRegistry.snapshot().size(), playerId);
                ctx.writeAndFlush(PacketCodec.encode(info) + "\n");
            } else if (packet instanceof PingPacket ping) {
                ctx.writeAndFlush(PacketCodec.encode(new PongPacket(ping.timestamp)) + "\n");
            } else if (packet instanceof PlayerUpdatePacket pu) {
                pu.state.id = playerId;
                pu.state.name = playerName;
                PlayerRegistry.update(ctx.channel(), pu.state);
                // Рассылку делает тикер LodeenServer — не здесь
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
