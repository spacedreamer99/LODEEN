package com.lodeen.engine.ui;

import imgui.ImGui;
import imgui.flag.ImGuiCond;
import imgui.flag.ImGuiWindowFlags;

public class PauseMenu {
    public enum Action { NONE, RESUME, EXIT_TO_MENU }

    public Action render(int w, int h) {
        ImGui.setNextWindowPos(w * 0.5f, h * 0.5f, ImGuiCond.Always, 0.5f, 0.5f);
        ImGui.setNextWindowSize(400, 260, ImGuiCond.Always);
        int flags = ImGuiWindowFlags.NoTitleBar | ImGuiWindowFlags.NoResize
                  | ImGuiWindowFlags.NoMove | ImGuiWindowFlags.NoCollapse;
        ImGui.begin("Pause", flags);
        ImGui.setCursorPosY(30);
        ImGui.setCursorPosX(140);
        ImGui.text("PAUSED");
        ImGui.spacing();
        ImGui.spacing();

        Action action = Action.NONE;
        if (ImGui.button("Resume", 320, 50))           action = Action.RESUME;
        ImGui.setCursorPosX(40);
        if (ImGui.button("Disconnect to Menu", 320, 50)) action = Action.EXIT_TO_MENU;
        ImGui.setCursorPosX(40);
        ImGui.button("Settings", 320, 50);

        ImGui.end();
        return action;
    }
}
