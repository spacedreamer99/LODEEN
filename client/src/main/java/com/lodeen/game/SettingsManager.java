package com.lodeen.game;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

public class SettingsManager {
    private static final Logger log = LoggerFactory.getLogger(SettingsManager.class);
    private static final Path CONFIG_DIR = Paths.get(System.getProperty("user.home"), ".lodeen");
    private static final Path CONFIG_FILE = CONFIG_DIR.resolve("settings.json");
    private static final ObjectMapper MAPPER = new ObjectMapper()
        .enable(SerializationFeature.INDENT_OUTPUT);

    private static Settings current = new Settings();
    private static boolean settingsVisible = false;

    public static Settings get() { return current; }

    public static boolean isSettingsVisible() { return settingsVisible; }
    public static void openSettings()   { settingsVisible = true; }
    public static void closeSettings()  { settingsVisible = false; }
    public static void toggleSettings() { settingsVisible = !settingsVisible; }

    public static void resetToDefaults() {
        current = new Settings();
        log.info("Settings reset to defaults");
    }

    public static void load() {
        try {
            if (Files.exists(CONFIG_FILE)) {
                current = MAPPER.readValue(CONFIG_FILE.toFile(), Settings.class);
                log.info("Settings loaded from {}", CONFIG_FILE);
            } else {
                log.info("No settings file, creating defaults: {}", CONFIG_FILE);
                save();
            }
        } catch (IOException e) {
            log.error("Failed to load settings, using defaults", e);
            current = new Settings();
        }
    }

    public static void save() {
        try {
            Files.createDirectories(CONFIG_DIR);
            MAPPER.writeValue(CONFIG_FILE.toFile(), current);
            log.debug("Settings saved to {}", CONFIG_FILE);
        } catch (IOException e) {
            log.error("Failed to save settings", e);
        }
    }
}
