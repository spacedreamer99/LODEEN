package com.lodeen.server.net;

import com.lodeen.server.ServerConfig;
import com.lodeen.shared.network.PacketCodec;
import com.lodeen.shared.network.WorldSnapshotPacket;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import io.netty.handler.codec.LineBasedFrameDecoder;
import io.netty.handler.codec.string.StringDecoder;
import io.netty.handler.codec.string.StringEncoder;
import io.netty.util.CharsetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

public class LodeenServer {
    private static final Logger log = LoggerFactory.getLogger(LodeenServer.class);
    private static final int TICK_HZ = 20;

    private final ServerConfig config;
    private EventLoopGroup bossGroup, workerGroup;
    private Channel serverChannel;
    private ScheduledExecutorService ticker;

    public LodeenServer(ServerConfig config) { this.config = config; }

    public void start() throws InterruptedException {
        bossGroup = new NioEventLoopGroup(1);
        workerGroup = new NioEventLoopGroup();

        ServerBootstrap b = new ServerBootstrap();
        b.group(bossGroup, workerGroup)
         .channel(NioServerSocketChannel.class)
         .option(ChannelOption.SO_BACKLOG, 128)
         .childOption(ChannelOption.SO_KEEPALIVE, true)
         .childHandler(new ChannelInitializer<SocketChannel>() {
             @Override protected void initChannel(SocketChannel ch) {
                 ch.pipeline()
                   .addLast(new LineBasedFrameDecoder(64 * 1024))
                   .addLast(new StringDecoder(CharsetUtil.UTF_8))
                   .addLast(new StringEncoder(CharsetUtil.UTF_8))
                   .addLast(new ServerHandler());
             }
         });

        serverChannel = b.bind(config.host, config.port).sync().channel();
        log.info("LODEEN Server listening on {}:{}", config.host, config.port);

        // Тикер — рассылает снапшот ровно TICK_HZ раз в секунду
        ticker = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "server-ticker");
            t.setDaemon(true);
            return t;
        });
        long periodMs = 1000L / TICK_HZ;
        ticker.scheduleAtFixedRate(this::broadcastTick, periodMs, periodMs, TimeUnit.MILLISECONDS);
    }

    private void broadcastTick() {
        try {
            int n = PlayerRegistry.snapshot().size();
            if (n == 0) return;
            WorldSnapshotPacket snap = new WorldSnapshotPacket(PlayerRegistry.snapshot());
            String json = PacketCodec.encode(snap) + "\n";
            PlayerRegistry.channels().writeAndFlush(json);
        } catch (Exception e) {
            log.error("Tick broadcast failed: {}", e.getMessage());
        }
    }

    public void stop() {
        log.info("Shutting down server...");
        if (ticker != null) ticker.shutdownNow();
        if (serverChannel != null) serverChannel.close();
        if (bossGroup != null) bossGroup.shutdownGracefully();
        if (workerGroup != null) workerGroup.shutdownGracefully();
    }

    public void await() throws InterruptedException {
        if (serverChannel != null) serverChannel.closeFuture().sync();
    }
}
