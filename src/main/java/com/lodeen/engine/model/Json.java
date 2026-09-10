package com.lodeen.engine.model;

import java.util.*;

public class Json {
    private final String s;
    private int i;
    private Json(String s) { this.s = s; }

    public static Object parse(String src) {
        Json p = new Json(src);
        p.ws();
        return p.value();
    }

    private Object value() {
        char c = s.charAt(i);
        if (c == '{') return object();
        if (c == '[') return array();
        if (c == '"') return string();
        if (c == 't') { i += 4; return Boolean.TRUE; }
        if (c == 'f') { i += 5; return Boolean.FALSE; }
        if (c == 'n') { i += 4; return null; }
        return number();
    }

    private Map<String,Object> object() {
        Map<String,Object> m = new LinkedHashMap<>();
        i++; ws();
        if (s.charAt(i) == '}') { i++; return m; }
        while (true) {
            ws(); String k = string(); ws(); i++; ws();
            m.put(k, value()); ws();
            if (s.charAt(i++) == '}') return m;
        }
    }

    private List<Object> array() {
        List<Object> a = new ArrayList<>();
        i++; ws();
        if (s.charAt(i) == ']') { i++; return a; }
        while (true) {
            ws(); a.add(value()); ws();
            if (s.charAt(i++) == ']') return a;
        }
    }

    private String string() {
        i++; StringBuilder sb = new StringBuilder();
        while (true) {
            char c = s.charAt(i++);
            if (c == '"') return sb.toString();
            if (c == '\\') {
                char e = s.charAt(i++);
                switch (e) {
                    case 'n': sb.append('\n'); break;
                    case 't': sb.append('\t'); break;
                    case 'r': sb.append('\r'); break;
                    case '"': sb.append('"'); break;
                    case '\\': sb.append('\\'); break;
                    case '/': sb.append('/'); break;
                    case 'u':
                        sb.append((char) Integer.parseInt(s.substring(i, i + 4), 16));
                        i += 4; break;
                    default: sb.append(e);
                }
            } else sb.append(c);
        }
    }

    private Double number() {
        int st = i;
        while (i < s.length() && "0123456789+-.eE".indexOf(s.charAt(i)) >= 0) i++;
        return Double.parseDouble(s.substring(st, i));
    }

    private void ws() {
        while (i < s.length() && Character.isWhitespace(s.charAt(i))) i++;
    }
}
