package com.lodeen.engine.scene;

import com.lodeen.engine.graphics.Renderer3D;
import com.lodeen.engine.graphics.Texture;
import com.lodeen.engine.model.ModelLoader;
import com.lodeen.engine.model.ObjectInstance;
import org.joml.Vector3f;
import java.util.*;
import static org.lwjgl.opengl.GL11.*;

public class Scene {
    private Camera camera;
    private InputController input;
    private DebugLogger debug;
    private Renderer3D renderer;
    private List<ObjectInstance> objects;
    private boolean exitRequested = false;

    public void init(long window) {
        camera = new Camera();
        renderer = new Renderer3D();
        objects = ModelLoader.load("/models/planet.glb");

        float radius = 0.001f;
        for (ObjectInstance oi : objects)
            radius = Math.max(radius, oi.mesh.getBoundingRadius());
        camera.fitToRadius(radius);
        input = new InputController(window, camera);
        debug = new DebugLogger(window, camera);
        System.out.println("Loaded objects: " + objects.size() + ", radius=" + radius);

        glEnable(GL_DEPTH_TEST);
        glEnable(GL_CULL_FACE);
        glCullFace(GL_BACK);
        glFrontFace(GL_CCW);
        glEnable(GL_BLEND);
        glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
        glClearColor(0.02f, 0.02f, 0.05f, 1f);
    }

    public void update(float dt) {
        if (dt > 0.1f) dt = 0.1f;
        if (dt <= 0) return;
        input.update(dt);
        debug.update(dt);
        if (input.isExitRequested()) exitRequested = true;
    }

    public void render(int w, int h) {
        glViewport(0, 0, w, h);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

        List<ObjectInstance> opaque = new ArrayList<>();
        List<ObjectInstance> transparent = new ArrayList<>();
        for (ObjectInstance oi : objects) {
            if (oi.mesh.material.alphaMode == 2) transparent.add(oi);
            else opaque.add(oi);
        }

        // Непрозрачные: обычная отрисовка с отсечением задних граней
        glCullFace(GL_BACK);
        glDepthMask(true);
        renderer.render(opaque, camera, w, h);

        // Прозрачные: два прохода — сначала задние стенки, потом передние.
        // Сортировка по расстоянию теряет смысл: обе половины одной сферы
        // корректно накладываются именно в этом порядке.
        glDepthMask(false);

        // 1) задние грани (то, что за сферой, включая дальнюю стенку для наблюдателя внутри)
        glCullFace(GL_FRONT);
        renderer.render(transparent, camera, w, h);

        // 2) передние грани (ближняя стенка, включая то, что видно изнутри)
        glCullFace(GL_BACK);
        renderer.render(transparent, camera, w, h);

        glDepthMask(true);
    }

    public boolean shouldExit() { return exitRequested; }

    public void cleanup() {
        if (input != null) input.captureMouse(false);
        Set<Texture> cleaned = Collections.newSetFromMap(new IdentityHashMap<>());
        for (ObjectInstance oi : objects) {
            oi.mesh.cleanup();
            Texture t = oi.mesh.material.baseColorTexture;
            if (t != null && cleaned.add(t)) t.cleanup();
        }
        renderer.cleanup();
        glDisable(GL_DEPTH_TEST);
        glDisable(GL_CULL_FACE);
    }
}
