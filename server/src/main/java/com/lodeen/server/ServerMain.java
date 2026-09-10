package com.lodeen.server;

import com.lodeen.server.db.Database;
import com.lodeen.server.db.PlayerDao;
import com.lodeen.server.db.PlayerRecord;
import com.lodeen.server.net.LodeenServer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import picocli.CommandLine;
import picocli.CommandLine.Command;
import picocli.CommandLine.Option;

import java.util.concurrent.Callable;

@Command(
    name = "lodeen-server",
    mixinStandardHelpOptions = true,
    version = "0.2.0",
    description = "LODEEN game server",
    subcommands = {
        ServerMain.StartCommand.class,
        ServerMain.InfoCommand.class,
        ServerMain.WhoCommand.class
    }
)
public class ServerMain implements Callable<Integer> {
    private static final Logger log = LoggerFactory.getLogger(ServerMain.class);

    @Override public Integer call() {
        CommandLine.usage(this, System.out);
        return 0;
    }

    public static void main(String[] args) {
        int exitCode = new CommandLine(new ServerMain()).execute(args);
        System.exit(exitCode);
    }

    @Command(name = "start", description = "Start the game server")
    static class StartCommand implements Callable<Integer> {
        @Option(names = {"-p", "--port"}) int port = 25565;
        @Option(names = {"-h", "--host"}) String host = "0.0.0.0";

        @Override public Integer call() throws Exception {
            Database.init();

            ServerConfig config = new ServerConfig();
            config.port = port;
            config.host = host;

            LodeenServer server = new LodeenServer(config);
            Runtime.getRuntime().addShutdownHook(new Thread(() -> {
                server.stop();
                Database.close();
            }));

            server.start();
            log.info("Server started. Press Ctrl+C to stop.");
            server.await();
            return 0;
        }
    }

    @Command(name = "info", description = "Show server info")
    static class InfoCommand implements Callable<Integer> {
        @Override public Integer call() {
            System.out.println("LODEEN Server v0.2.0");
            System.out.println("  Java:   " + System.getProperty("java.version"));
            System.out.println("  OS:     " + System.getProperty("os.name"));
            System.out.println("  Cores:  " + Runtime.getRuntime().availableProcessors());
            System.out.println("  Memory: " + (Runtime.getRuntime().maxMemory() / 1024 / 1024) + " MB max");
            return 0;
        }
    }

    @Command(name = "who", description = "List all registered players")
    static class WhoCommand implements Callable<Integer> {
        @Override public Integer call() {
            Database.init();
            var list = PlayerDao.all();
            if (list.isEmpty()) {
                System.out.println("No players registered.");
            } else {
                System.out.println("Registered players: " + list.size());
                System.out.printf("  %-18s %-20s %-12s %-12s%n",
                    "ID", "NAME", "REGISTERED", "PLAYTIME");
                for (PlayerRecord p : list) {
                    System.out.printf("  %-18s %-20s %-12s %-12s%n",
                        p.id(), p.name(),
                        timeAgo(p.registeredAt()),
                        p.playtimeSec() + "s");
                }
            }
            Database.close();
            return 0;
        }

        private static String timeAgo(long epochSec) {
            long diff = System.currentTimeMillis() / 1000L - epochSec;
            if (diff < 60) return diff + "s ago";
            if (diff < 3600) return (diff / 60) + "m ago";
            if (diff < 86400) return (diff / 3600) + "h ago";
            return (diff / 86400) + "d ago";
        }
    }
}
