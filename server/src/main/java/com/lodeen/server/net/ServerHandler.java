package com.lodeen.server.net;

import com.lodeen.shared.network.Packet;
import com.lodeen.shared.network.PacketCodec;
import com.lodeen.shared.network.PingPacket;
import com.lodeen.shared.network.PongPacket;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ServerHandler extends SimpleChannelInboundHandler<String> {
    private static final Logger log = LoggerFactory.getLogger(ServerHandler.class);

    @Override
    public void channelActive(ChannelHandlerContext ctx) {
        log.info("Client connected: {}", ctx.channel().remoteAddress());
    }

    @Override
    public void channelInactive(ChannelHandlerContext ctx) {
        log.info("Client disconnected: {}", ctx.channel().remoteAddress());
    }

    @Override
    protected void channelRead0(ChannelHandlerContext ctx, String msg) {
        try {
            Packet packet = PacketCodec.decode(msg);
            log.debug("Received packet: {} from {}", packet.id(), ctx.channel().remoteAddress());

            if (packet instanceof PingPacket ping) {
                PongPacket pong = new PongPacket(ping.timestamp);
                ctx.writeAndFlush(PacketCodec.encode(pong) + "\n");
                log.debug("Replied pong to {}", ctx.channel().remoteAddress());
            } else {
                log.warn("Unhandled packet: {}", packet.id());
            }
        } catch (Exception e) {
            log.error("Failed to process packet: {}", e.getMessage());
        }
    }

    @Override
    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
        log.error("Channel error: {}", cause.getMessage());
        ctx.close();
    }
}
