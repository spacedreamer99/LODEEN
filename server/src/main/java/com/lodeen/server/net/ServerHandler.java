package com.lodeen.server.net;

import com.lodeen.server.db.PlayerDao;
import com.lodeen.server.db.PlayerRecord;
import com.lodeen.shared.network.*;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ServerHandler extends SimpleChannelInboundHandler<String> {
    private static final Logger log = LoggerFactory.getLogger(ServerHandler.class);

    private String playerName = "unknown";
    private String playerId;
    private long connectedAt;

    @Override public void channelActive(ChannelHandlerContext ctx) {
        connectedAt = System.currentTimeMillis();
        log.info("Client connected: {}", ctx.channel().remoteAddress());
    }

    @Override public void channelInactive(ChannelHandlerContext ctx) {
        PlayerRegistry.remove(ctx.channel());
        if (playerId != null) {
            long sessionSec = (System.currentTimeMillis() - connectedAt) / 1000L;
            PlayerDao.updateLastSeen(playerId, sessionSec);
        }
        log.info("Client disconnected: {} (player: {})", ctx.channel().remoteAddress(), playerName);
    }

    @Override protected void channelRead0(ChannelHandlerContext ctx, String msg) {
        try {
            Packet packet = PacketCodec.decode(msg);

            if (packet instanceof HandshakePacket hs) {
                this.playerName = hs.playerName;
                PlayerRecord rec = PlayerDao.findOrCreate(playerName);
                if (rec == null) {
                    log.warn("DB registration failed for {}", playerName);
                    this.playerId = ctx.channel().id().asShortText();
                } else {
                    this.playerId = rec.id();
                    log.info("Player {} authenticated (db_id={}, playtime={}s)",
                        playerName, playerId, rec.playtimeSec());
                }
                PlayerState st = new PlayerState(playerId, playerName, 0, 0, 3, 0, 0);
                PlayerRegistry.add(ctx.channel(), st);

                ServerInfoPacket info = new ServerInfoPacket("0.2.0", "LODEEN dev server",
                        PlayerRegistry.snapshot().size(), playerId);
                ctx.writeAndFlush(PacketCodec.encode(info) + "\n");
            } else if (packet instanceof PingPacket ping) {
                ctx.writeAndFlush(PacketCodec.encode(new PongPacket(ping.timestamp)) + "\n");
            } else if (packet instanceof PlayerUpdatePacket pu) {
                pu.state.id = playerId;
                pu.state.name = playerName;
                PlayerRegistry.update(ctx.channel(), pu.state);
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
