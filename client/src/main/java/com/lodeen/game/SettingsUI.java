package com.lodeen.game;

import imgui.ImGui;
import imgui.flag.ImGuiCond;
import imgui.flag.ImGuiWindowFlags;

public class SettingsUI {

    private static boolean dirty = false;

    public static void render(int screenW, int screenH) {
        Settings s = SettingsManager.get();

        ImGui.setNextWindowPos(screenW * 0.5f, screenH * 0.5f, ImGuiCond.Appearing, 0.5f, 0.5f);
        ImGui.setNextWindowSize(520, 600, ImGuiCond.Appearing);

        ImGui.begin("Settings", ImGuiWindowFlags.NoCollapse);

        ImGui.text("Controls");
        ImGui.separator();

        dirty |= sliderFloat("Mouse sensitivity", s.mouseSensitivity, 0.01f, 1.0f, "%.3f");
        s.mouseSensitivity = readLastFloat();
        dirty |= sliderFloat("Linear speed", s.linearSpeed, 0.5f, 50.0f, "%.1f");
        s.linearSpeed = readLastFloat();
        dirty |= sliderFloat("Angular speed", s.angularSpeed, 10.0f, 300.0f, "%.0f");
        s.angularSpeed = readLastFloat();

        if (ImGui.checkbox("Invert Y", s.invertY)) dirty = true;

        ImGui.spacing();
        ImGui.text("Camera");
        ImGui.separator();

        dirty |= sliderFloat("FOV", s.fov, 30.0f, 120.0f, "%.0f");
        s.fov = readLastFloat();
        dirty |= sliderFloat("Near", s.near, 0.001f, 1.0f, "%.4f");
        s.near = readLastFloat();
        dirty |= sliderFloat("Far", s.far, 100.0f, 10000.0f, "%.0f");
        s.far = readLastFloat();

        ImGui.spacing();
        ImGui.text("Graphics");
        ImGui.separator();

        if (ImGui.checkbox("VSync", s.vsync)) dirty = true;
        if (ImGui.checkbox("Show debug overlay (F3)", s.showDebugOverlay)) dirty = true;

        ImGui.spacing();
        ImGui.separator();

        if (ImGui.button("Save", 120, 35)) {
            SettingsManager.save();
            dirty = false;
        }
        ImGui.sameLine();
        if (ImGui.button("Reset to defaults", 180, 35)) {
            SettingsManager.resetToDefaults();
            dirty = true;
        }
        ImGui.sameLine();
        if (ImGui.button("Close", 120, 35)) {
            if (dirty) SettingsManager.save();
            SettingsManager.closeSettings();
        }

        ImGui.end();
    }

    private static final float[] buffer = new float[1];

    private static boolean sliderFloat(String label, float value, float min, float max, String fmt) {
        buffer[0] = value;
        return ImGui.sliderFloat(label, buffer, min, max, fmt);
    }

    private static float readLastFloat() {
        return buffer[0];
    }
}
