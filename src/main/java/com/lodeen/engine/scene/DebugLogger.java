package com.lodeen.engine.scene;

import org.joml.Vector3f;
import static org.lwjgl.glfw.GLFW.*;

public class DebugLogger {
    private final long window;
    private final Camera camera;
    private double timer = 0;
    private static final double INTERVAL = 0.25;

    public DebugLogger(long window, Camera camera) {
        this.window = window;
        this.camera = camera;
    }

    public void update(float dt) {
        timer += dt;
        if (timer < INTERVAL) return;
        timer = 0;

        Vector3f p = camera.getPosition();
        Vector3f f = camera.forward();
        Vector3f u = camera.up();
        Vector3f r = camera.right();

        StringBuilder keys = new StringBuilder();
        check(keys, GLFW_KEY_W, "W");
        check(keys, GLFW_KEY_A, "A");
        check(keys, GLFW_KEY_S, "S");
        check(keys, GLFW_KEY_D, "D");
        check(keys, GLFW_KEY_SPACE, "SPACE");
        check(keys, GLFW_KEY_LEFT_SHIFT, "SHIFT");
        check(keys, GLFW_KEY_UP, "UP");
        check(keys, GLFW_KEY_DOWN, "DOWN");
        check(keys, GLFW_KEY_LEFT, "LEFT");
        check(keys, GLFW_KEY_RIGHT, "RIGHT");
        check(keys, GLFW_KEY_Q, "Q");
        check(keys, GLFW_KEY_E, "E");

        System.out.printf(
            "[DBG] pos=(%6.2f,%6.2f,%6.2f) " +
            "fwd=(%5.2f,%5.2f,%5.2f) " +
            "up=(%5.2f,%5.2f,%5.2f) " +
            "right=(%5.2f,%5.2f,%5.2f) keys=[%s]%n",
            p.x, p.y, p.z,
            f.x, f.y, f.z,
            u.x, u.y, u.z,
            r.x, r.y, r.z, keys);
    }

    private void check(StringBuilder sb, int key, String name) {
        if (glfwGetKey(window, key) == GLFW_PRESS) {
            if (sb.length() > 0) sb.append(' ');
            sb.append(name);
        }
    }
}
