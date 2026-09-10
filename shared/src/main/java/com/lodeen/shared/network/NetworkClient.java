package com.lodeen.shared.network;

import io.netty.bootstrap.Bootstrap;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioSocketChannel;
import io.netty.handler.codec.LineBasedFrameDecoder;
import io.netty.handler.codec.string.StringDecoder;
import io.netty.handler.codec.string.StringEncoder;
import io.netty.util.CharsetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class NetworkClient {
    private static final Logger log = LoggerFactory.getLogger(NetworkClient.class);

    public interface Listener {
        void onConnected();
        void onDisconnected();
        void onPacket(Packet packet);
    }

    private final String host;
    private final int port;
    private final Listener listener;
    private EventLoopGroup group;
    private Channel channel;
    private volatile boolean connected = false;
    private volatile long lastPingSent = 0;
    private volatile long lastPingMs = -1;

    public NetworkClient(String host, int port, Listener listener) {
        this.host = host; this.port = port; this.listener = listener;
    }

    public void connect() { new Thread(this::connectBlocking, "network-client").start(); }

    private void connectBlocking() {
        group = new NioEventLoopGroup(1);
        try {
            Bootstrap b = new Bootstrap();
            b.group(group).channel(NioSocketChannel.class)
             .option(ChannelOption.SO_KEEPALIVE, true)
             .handler(new ChannelInitializer<SocketChannel>() {
                 @Override protected void initChannel(SocketChannel ch) {
                     ch.pipeline()
                       .addLast(new LineBasedFrameDecoder(64 * 1024))
                       .addLast(new StringDecoder(CharsetUtil.UTF_8))
                       .addLast(new StringEncoder(CharsetUtil.UTF_8))
                       .addLast(new ClientHandler());
                 }
             });
            channel = b.connect(host, port).sync().channel();
            connected = true;
            log.info("Connected to {}:{}", host, port);
            if (listener != null) listener.onConnected();
            channel.closeFuture().sync();
        } catch (Exception e) {
            log.warn("Connection failed: {}", e.getMessage());
        } finally {
            connected = false;
            if (listener != null) listener.onDisconnected();
            group.shutdownGracefully();
        }
    }

    public boolean isConnected() { return connected; }
    public long getLastPingMs() { return lastPingMs; }

    public void send(Packet packet) {
        if (!connected || channel == null || !channel.isActive()) return;
        try { channel.writeAndFlush(PacketCodec.encode(packet) + "\n"); }
        catch (Exception e) { log.error("Send failed: {}", e.getMessage()); }
    }

    public void tickPing() {
        if (!connected) return;
        long now = System.currentTimeMillis();
        if (now - lastPingSent > 1000) { lastPingSent = now; send(new PingPacket(now)); }
    }

    public void disconnect() {
        if (channel != null) channel.close();
        if (group != null) group.shutdownGracefully();
    }

    private class ClientHandler extends SimpleChannelInboundHandler<String> {
        @Override protected void channelRead0(ChannelHandlerContext ctx, String msg) {
            try {
                Packet p = PacketCodec.decode(msg);
                if (p instanceof PongPacket pong) lastPingMs = System.currentTimeMillis() - pong.timestamp;
                if (listener != null) listener.onPacket(p);
            } catch (Exception e) { log.error("Decode failed: {}", e.getMessage()); }
        }
        @Override public void exceptionCaught(ChannelHandlerContext ctx, Throwable c) { ctx.close(); }
    }
}
