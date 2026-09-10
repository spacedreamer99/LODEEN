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
    private String sessionId;   // уникален для каждого подключения
    private String dbId;         // постоянный ID в БД
    private long connectedAt;

    @Override public void channelActive(ChannelHandlerContext ctx) {
        connectedAt = System.currentTimeMillis();
        log.info("Client connected: {}", ctx.channel().remoteAddress());
    }

    @Override public void channelInactive(ChannelHandlerContext ctx) {
        PlayerRegistry.remove(ctx.channel());
        if (dbId != null) {
            long sessionSec = (System.currentTimeMillis() - connectedAt) / 1000L;
            PlayerDao.updateLastSeen(dbId, sessionSec);
        }
        log.info("Client disconnected: {} (player: {})", ctx.channel().remoteAddress(), playerName);
    }

    @Override protected void channelRead0(ChannelHandlerContext ctx, String msg) {
        try {
            Packet packet = PacketCodec.decode(msg);

            if (packet instanceof HandshakePacket hs) {
                this.playerName = hs.playerName;
                PlayerRecord rec = PlayerDao.findOrCreate(playerName);
                this.dbId = (rec != null) ? rec.id() : "unknown";
                this.sessionId = dbId + ":" + ctx.channel().id().asShortText();

                log.info("Player {} authenticated (db_id={}, session={})",
                    playerName, dbId, sessionId);

                float angle = (float) (Math.random() * Math.PI * 2);
                float radius = 3.5f;
                float sx = (float) Math.cos(angle) * radius;
                float sz = (float) Math.sin(angle) * radius;
                PlayerState st = new PlayerState(sessionId, playerName, sx, 0, sz, 0, 0);
                PlayerRegistry.add(ctx.channel(), st);

                ServerInfoPacket info = new ServerInfoPacket("0.2.0", "LODEEN dev server",
                        PlayerRegistry.snapshot().size(), sessionId);
                ctx.writeAndFlush(PacketCodec.encode(info) + "\n");
            } else if (packet instanceof PingPacket ping) {
                ctx.writeAndFlush(PacketCodec.encode(new PongPacket(ping.timestamp)) + "\n");
            } else if (packet instanceof PlayerUpdatePacket pu) {
                pu.state.id = sessionId;
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
