package com.lodeen.server.net;

import com.lodeen.server.ServerConfig;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import io.netty.handler.codec.string.StringDecoder;
import io.netty.handler.codec.string.StringEncoder;
import io.netty.handler.codec.LineBasedFrameDecoder;
import io.netty.util.CharsetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class LodeenServer {
    private static final Logger log = LoggerFactory.getLogger(LodeenServer.class);

    private final ServerConfig config;
    private EventLoopGroup bossGroup;
    private EventLoopGroup workerGroup;
    private Channel serverChannel;

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
             @Override
             protected void initChannel(SocketChannel ch) {
                 ChannelPipeline p = ch.pipeline();
                 p.addLast(new LineBasedFrameDecoder(64 * 1024));
                 p.addLast(new StringDecoder(CharsetUtil.UTF_8));
                 p.addLast(new StringEncoder(CharsetUtil.UTF_8));
                 p.addLast(new ServerHandler());
             }
         });

        serverChannel = b.bind(config.host, config.port).sync().channel();
        log.info("LODEEN Server listening on {}:{}", config.host, config.port);
    }

    public void stop() {
        log.info("Shutting down server...");
        if (serverChannel != null) serverChannel.close();
        if (bossGroup != null) bossGroup.shutdownGracefully();
        if (workerGroup != null) workerGroup.shutdownGracefully();
    }

    public void await() throws InterruptedException {
        if (serverChannel != null) serverChannel.closeFuture().sync();
    }
}
