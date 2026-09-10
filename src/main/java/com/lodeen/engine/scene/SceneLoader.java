package com.lodeen.engine.scene;

import com.lodeen.engine.model.ModelLoader;
import java.util.List;

public class SceneLoader {
    public static class Loaded {
        public List<GameObject> objects;
        public float planetRadius;
    }

    public static Loaded load(String path) {
        Loaded l = new Loaded();
        l.objects = ModelLoader.load(path);
        l.planetRadius = 0.001f;
        for (GameObject go : l.objects)
            l.planetRadius = Math.max(l.planetRadius, go.boundingRadius());
        return l;
    }
}
