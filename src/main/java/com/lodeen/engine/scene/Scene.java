package com.lodeen.engine.scene;

import com.lodeen.engine.graphics.Renderer3D;
import com.lodeen.engine.graphics.SkyboxRenderer;
import com.lodeen.engine.graphics.Texture;
import java.util.*;
import static org.lwjgl.opengl.GL11.*;

public class Scene {
    private Camera camera;
    private InputController input;
    private DebugLogger debug;
    private Renderer3D renderer;
    private SkyboxRenderer skybox;
    private SceneRenderer sceneRenderer;
    private List<GameObject> objects;
    private boolean exitRequested = false;

    public void init(long window) {
        camera = new Camera();
        renderer = new Renderer3D();
        skybox = new SkyboxRenderer();

        SceneLoader.Loaded loaded = SceneLoader.load("/models/planet.glb");
        objects = loaded.objects;
        camera.fitToRadius(loaded.planetRadius);
        System.out.println("Planet R=" + loaded.planetRadius);

        sceneRenderer = new SceneRenderer(renderer, skybox);
        input = new InputController(window, camera);
        debug = new DebugLogger(window, camera);
        glEnable(GL_DEPTH_TEST);
        glEnable(GL_CULL_FACE);
        glCullFace(GL_BACK);
        glFrontFace(GL_CCW);
        glClearColor(0, 0, 0, 1);
    }

    public void update(float dt) {
        if (dt > 0.1f) dt = 0.1f;
        if (dt <= 0) return;
        input.update(dt);
        debug.update(dt);
        if (input.isExitRequested()) exitRequested = true;
    }

    public void render(int w, int h) {
        sceneRenderer.render(objects, camera, w, h);
    }

    public boolean shouldExit() { return exitRequested; }

    public void cleanup() {
        if (input != null) input.captureMouse(false);
        Set<Texture> done = Collections.newSetFromMap(new IdentityHashMap<>());
        for (GameObject go : objects) {
            if (go.mesh == null) continue;
            go.mesh.cleanup();
            Texture a = go.mesh.material.baseColorTexture;
            Texture b = go.mesh.material.metallicRoughnessTexture;
            if (a != null && done.add(a)) a.cleanup();
            if (b != null && done.add(b)) b.cleanup();
        }
        skybox.cleanup();
        renderer.cleanup();
        glDisable(GL_DEPTH_TEST);
        glDisable(GL_CULL_FACE);
    }
}
