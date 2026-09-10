package com.lodeen;

import com.lodeen.engine.core.Window;

public class Main {
    public static void main(String[] args) {
        Window window = new Window("LODEEN Engine", 1280, 720);
        window.init();
        window.loop();
    }
}
