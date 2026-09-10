package com.lodeen.engine.model;

import java.nio.*;
import java.nio.charset.StandardCharsets;
import java.util.*;

public class GlbLoader {
    public final Map<String,Object> json;
    public final ByteBuffer bin;
    private final List<Map<String,Object>> accessors;
    private final List<Map<String,Object>> bufferViews;

    @SuppressWarnings("unchecked")
    public GlbLoader(byte[] data) {
        ByteBuffer buf = ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
        if (buf.getInt() != 0x46546C67) throw new RuntimeException("Not a GLB file");
        buf.getInt(); buf.getInt();
        String jsonStr = null; ByteBuffer binBuf = null;
        while (buf.remaining() >= 8) {
            int len = buf.getInt();
            int type = buf.getInt();
            int next = buf.position() + len;
            if (type == 0x4E4F534A) {
                byte[] jb = new byte[len]; buf.get(jb);
                jsonStr = new String(jb, StandardCharsets.UTF_8);
            } else if (type == 0x004E4942) {
                byte[] bb = new byte[len]; buf.get(bb);
                binBuf = ByteBuffer.wrap(bb).order(ByteOrder.LITTLE_ENDIAN);
            }
            buf.position(next);
        }
        json = (Map<String,Object>) Json.parse(jsonStr);
        bin = binBuf;
        accessors = (List<Map<String,Object>>) json.get("accessors");
        bufferViews = (List<Map<String,Object>>) json.get("bufferViews");
    }

    public Map<String,Object> accessor(int i) { return accessors.get(i); }
    public Map<String,Object> bufferView(int i) { return bufferViews.get(i); }

    public float[] readFloats(int accIdx) {
        Map<String,Object> acc = accessors.get(accIdx);
        int count = num(acc, "count");
        int comps = components((String) acc.get("type"));
        ByteBuffer bb = slice(acc);
        float[] out = new float[count * comps];
        int stride = stride(acc, comps * 4);
        if (stride < comps * 4) stride = comps * 4;
        for (int i = 0; i < count; i++) {
            for (int c = 0; c < comps; c++) out[i*comps + c] = bb.getFloat();
            if (i < count - 1) bb.position(bb.position() + stride - comps * 4);
        }
        return out;
    }

    public int[] readInts(int accIdx) {
        Map<String,Object> acc = accessors.get(accIdx);
        int count = num(acc, "count");
        int compType = num(acc, "componentType");
        ByteBuffer bb = slice(acc);
        int[] out = new int[count];
        for (int i = 0; i < count; i++) {
            if (compType == 5125) out[i] = bb.getInt();
            else if (compType == 5123) out[i] = bb.getShort() & 0xFFFF;
            else if (compType == 5121) out[i] = bb.get() & 0xFF;
        }
        return out;
    }

    private ByteBuffer slice(Map<String,Object> acc) {
        int bvIdx = num(acc, "bufferView");
        int off = acc.containsKey("byteOffset") ? num(acc, "byteOffset") : 0;
        Map<String,Object> bv = bufferViews.get(bvIdx);
        int bvOff = bv.containsKey("byteOffset") ? num(bv, "byteOffset") : 0;
        ByteBuffer bb = bin.duplicate().order(ByteOrder.LITTLE_ENDIAN);
        bb.position(bvOff + off);
        return bb;
    }

    private int stride(Map<String,Object> acc, int fallback) {
        Map<String,Object> bv = bufferViews.get(num(acc, "bufferView"));
        return bv.containsKey("byteStride") ? num(bv, "byteStride") : fallback;
    }

    private static int num(Map<String,Object> m, String k) {
        return ((Number) m.get(k)).intValue();
    }
    private static int components(String type) {
        switch (type) {
            case "SCALAR": return 1;
            case "VEC2": return 2;
            case "VEC3": return 3;
            case "VEC4": return 4;
            case "MAT4": return 16;
            default: throw new RuntimeException("Unknown accessor type: " + type);
        }
    }
}
