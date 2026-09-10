package com.lodeen.server.db;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.List;

public class PlayerDao {
    private static final Logger log = LoggerFactory.getLogger(PlayerDao.class);

    /** Найти игрока по имени. Если нет — создать. Возвращает запись. */
    public static PlayerRecord findOrCreate(String name) {
        try (Connection c = Database.getConnection()) {
            PlayerRecord existing = findByName(c, name);
            if (existing != null) {
                touchLastSeen(c, existing.id());
                return existing;
            }
            PlayerRecord fresh = PlayerRecord.newPlayer(null, name);
            insert(c, fresh);
            return findByName(c, name);
        } catch (SQLException e) {
            log.error("findOrCreate failed: {}", e.getMessage());
            return null;
        }
    }

    /** Обновить время последнего визита + накопленное игровое время. */
    public static void updateLastSeen(String id, long playtimeSec) {
        String sql = "UPDATE players SET last_seen=?, playtime_sec=? WHERE id=?";
        try (Connection c = Database.getConnection();
             PreparedStatement ps = c.prepareStatement(sql)) {
            ps.setLong(1, System.currentTimeMillis() / 1000L);
            ps.setLong(2, playtimeSec);
            ps.setString(3, id);
            ps.executeUpdate();
        } catch (SQLException e) {
            log.error("updateLastSeen failed: {}", e.getMessage());
        }
    }

    /** Список всех игроков (для /who). */
    public static List<PlayerRecord> all() {
        List<PlayerRecord> out = new ArrayList<>();
        try (Connection c = Database.getConnection();
             PreparedStatement ps = c.prepareStatement(
                 "SELECT id, name, registered_at, last_seen, playtime_sec FROM players ORDER BY last_seen DESC");
             ResultSet rs = ps.executeQuery()) {
            while (rs.next()) out.add(map(rs));
        } catch (SQLException e) {
            log.error("all() failed: {}", e.getMessage());
        }
        return out;
    }

    private static PlayerRecord findByName(Connection c, String name) throws SQLException {
        try (PreparedStatement ps = c.prepareStatement(
                "SELECT id, name, registered_at, last_seen, playtime_sec FROM players WHERE name=?")) {
            ps.setString(1, name);
            try (ResultSet rs = ps.executeQuery()) {
                return rs.next() ? map(rs) : null;
            }
        }
    }

    private static void insert(Connection c, PlayerRecord p) throws SQLException {
        String sql = "INSERT INTO players(id, name, registered_at, last_seen, playtime_sec) " +
                     "VALUES (lower(hex(randomblob(8))), ?, ?, ?, ?)";
        try (PreparedStatement ps = c.prepareStatement(sql)) {
            ps.setString(1, p.name());
            ps.setLong(2, p.registeredAt());
            ps.setLong(3, p.lastSeen());
            ps.setLong(4, p.playtimeSec());
            ps.executeUpdate();
        }
    }

    private static void touchLastSeen(Connection c, String id) throws SQLException {
        try (PreparedStatement ps = c.prepareStatement(
                "UPDATE players SET last_seen=? WHERE id=?")) {
            ps.setLong(1, System.currentTimeMillis() / 1000L);
            ps.setString(2, id);
            ps.executeUpdate();
        }
    }

    private static PlayerRecord map(ResultSet rs) throws SQLException {
        return new PlayerRecord(
            rs.getString("id"),
            rs.getString("name"),
            rs.getLong("registered_at"),
            rs.getLong("last_seen"),
            rs.getLong("playtime_sec")
        );
    }
}
