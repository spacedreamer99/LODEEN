package com.lodeen.engine.scene;

import com.lodeen.game.SettingsManager;
import org.joml.Matrix4f;
import org.joml.Quaternionf;
import org.joml.Vector3f;

public class Camera {
    private final Vector3f position = new Vector3f(0, 0, 3);
    private final Quaternionf rotation = new Quaternionf();
    private float fov, near, far;
    private float linearSpeed;
    private float angularSpeed;
    private float throttle = 1f;

    public Camera() {
        applySettings();
    }

    public void applySettings() {
        var s = SettingsManager.get();
        this.fov = s.fov;
        this.near = s.near;
        this.far = s.far;
        this.linearSpeed = s.linearSpeed;
        this.angularSpeed = s.angularSpeed;
    }

    public Matrix4f getView() {
        Quaternionf inv = new Quaternionf(rotation).conjugate();
        return new Matrix4f().rotate(inv)
            .translate(-position.x, -position.y, -position.z);
    }

    public Matrix4f getProjection(int w, int h) {
        return new Matrix4f().perspective((float) java.lang.Math.toRadians(fov),
                java.lang.Math.max(0.01f, (float) w / h), near, far);
    }

    public Vector3f getPosition() { return position; }
    public Quaternionf getRotation() { return rotation; }
    public float getThrottle() { return throttle; }
    public float getFov() { return fov; }

    public Vector3f forward() { return new Vector3f(0, 0, -1).rotate(rotation); }
    public Vector3f right()   { return new Vector3f(1, 0, 0).rotate(rotation); }
    public Vector3f up()      { return new Vector3f(0, 1, 0).rotate(rotation); }

    public void fitToRadius(float r) {
        position.set(0, 0, java.lang.Math.max(1.5f, r * 3f));
        far = java.lang.Math.max(100f, r * 50f);
        near = java.lang.Math.max(0.001f, r * 0.001f);
    }

    public void multiplyThrottle(float factor) {
        throttle = Math.max(0.001f, Math.min(10f, throttle * factor));
    }

    public void update(float dt,
                       float fwdIn, float rgtIn, float upIn,
                       float pitchIn, float yawIn, float rollIn) {
        float lin = linearSpeed * throttle * dt;
        position.fma(fwdIn * lin, forward())
                .fma(rgtIn * lin, right())
                .fma(upIn  * lin, up());

        float ang = angularSpeed * dt;
        rotateLocal(pitchIn * ang, yawIn * ang, rollIn * ang);
    }

    public void rotateLocal(float pitchDeg, float yawDeg, float rollDeg) {
        if (yawDeg != 0)   rotateAroundLocalAxis(0, 1, 0, yawDeg);
        if (pitchDeg != 0) rotateAroundLocalAxis(1, 0, 0, pitchDeg);
        if (rollDeg != 0)  rotateAroundLocalAxis(0, 0, -1, rollDeg);
        rotation.normalize();
    }

    private void rotateAroundLocalAxis(float lx, float ly, float lz, float deg) {
        Vector3f worldAxis = new Vector3f(lx, ly, lz).rotate(rotation).normalize();
        Quaternionf q = new Quaternionf()
            .fromAxisAngleRad(worldAxis, (float) java.lang.Math.toRadians(deg));
        rotation.premul(q);
    }

    public void mouseLook(float dx, float dy) {
        rotateLocal(-dy, -dx, 0);
    }

    public void resetRotation() { rotation.identity(); }
}
