package com.lodeen.engine.ui;

import com.lodeen.client.net.NetworkManager;
import imgui.ImGui;
import imgui.flag.ImGuiCond;
import static org.lwjgl.glfw.GLFW.*;

public class DebugOverlay {
    private boolean visible = false;
    private boolean f3WasPressed = false;

    public void toggle(long window) {
        boolean pressed = glfwGetKey(window, GLFW_KEY_F3) == GLFW_PRESS;
        if (pressed && !f3WasPressed) visible = !visible;
        f3WasPressed = pressed;
    }

    public void render(float fps, String state, int w, int h, NetworkManager net) {
        if (!visible) return;
        ImGui.setNextWindowPos(10, 10, ImGuiCond.Always);
        ImGui.setNextWindowSize(300, 180, ImGuiCond.Always);
        ImGui.begin("Debug (F3)");
        ImGui.text(String.format("FPS: %.0f", fps));
        ImGui.text("State: " + state);
        ImGui.text("Res: " + w + " x " + h);
        if (net != null) {
            ImGui.separator();
            ImGui.text("Server: " + (net.isConnected() ? "Connected" : "Disconnected"));
            ImGui.text("Ping: " + (net.getPingMs() >= 0 ? net.getPingMs() + " ms" : "-"));
            ImGui.text("MOTD: " + net.getMotd());
        }
        ImGui.end();
    }
}
