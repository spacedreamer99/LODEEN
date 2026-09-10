package com.lodeen.server.db;

import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import org.flywaydb.core.Flyway;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.sql.DataSource;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.sql.Connection;
import java.sql.SQLException;

public class Database {
    private static final Logger log = LoggerFactory.getLogger(Database.class);
    private static HikariDataSource ds;

    public static void init() {
        try {
            Path dir = Paths.get(System.getProperty("user.home"), ".lodeen");
            Files.createDirectories(dir);
            Path dbFile = dir.resolve("lodeen.db");

            HikariConfig cfg = new HikariConfig();
            cfg.setJdbcUrl("jdbc:sqlite:" + dbFile.toAbsolutePath());
            cfg.setMaximumPoolSize(4);
            cfg.setPoolName("lodeen-db");
            cfg.setConnectionTimeout(10_000);

            ds = new HikariDataSource(cfg);

            Flyway flyway = Flyway.configure()
                .dataSource(ds)
                .locations("classpath:db/migration")
                .load();
            flyway.migrate();

            log.info("Database ready: {}", dbFile.toAbsolutePath());
        } catch (Exception e) {
            throw new RuntimeException("DB init failed", e);
        }
    }

    public static DataSource dataSource() { return ds; }

    public static Connection getConnection() throws SQLException { return ds.getConnection(); }

    public static void close() {
        if (ds != null) ds.close();
    }
}
