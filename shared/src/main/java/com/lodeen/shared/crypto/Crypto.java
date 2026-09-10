package com.lodeen.shared.crypto;

import org.bouncycastle.crypto.digests.SHA256Digest;
import org.bouncycastle.crypto.engines.AESEngine;
import org.bouncycastle.crypto.modes.GCMBlockCipher;
import org.bouncycastle.crypto.params.AEADParameters;
import org.bouncycastle.crypto.params.KeyParameter;
import org.bouncycastle.jce.provider.BouncyCastleProvider;

import javax.crypto.KeyAgreement;
import java.security.*;
import java.security.spec.ECGenParameterSpec;

public class Crypto {
    static {
        if (Security.getProvider(BouncyCastleProvider.PROVIDER_NAME) == null) {
            Security.addProvider(new BouncyCastleProvider());
        }
    }

    /** AES-GCM шифрование. Возвращает [nonce(12) + ciphertext + tag(16)] */
    public static byte[] encryptAesGcm(byte[] key, byte[] plaintext) throws Exception {
        byte[] nonce = new byte[12];
        new SecureRandom().nextBytes(nonce);

        GCMBlockCipher cipher = new GCMBlockCipher(new AESEngine());
        AEADParameters params = new AEADParameters(new KeyParameter(key), 128, nonce);
        cipher.init(true, params);

        byte[] output = new byte[cipher.getOutputSize(plaintext.length)];
        int len = cipher.processBytes(plaintext, 0, plaintext.length, output, 0);
        cipher.doFinal(output, len);

        byte[] result = new byte[nonce.length + output.length];
        System.arraycopy(nonce, 0, result, 0, nonce.length);
        System.arraycopy(output, 0, result, nonce.length, output.length);
        return result;
    }

    /** AES-GCM расшифровка. Вход: [nonce(12) + ciphertext + tag(16)] */
    public static byte[] decryptAesGcm(byte[] key, byte[] encrypted) throws Exception {
        byte[] nonce = new byte[12];
        System.arraycopy(encrypted, 0, nonce, 0, 12);

        GCMBlockCipher cipher = new GCMBlockCipher(new AESEngine());
        AEADParameters params = new AEADParameters(new KeyParameter(key), 128, nonce);
        cipher.init(false, params);

        byte[] output = new byte[cipher.getOutputSize(encrypted.length - 12)];
        int len = cipher.processBytes(encrypted, 12, encrypted.length - 12, output, 0);
        cipher.doFinal(output, len);
        return output;
    }

    /** Генерация пары ECDH-ключей (secp256r1) */
    public static KeyPair generateEcdhKeyPair() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance("EC", BouncyCastleProvider.PROVIDER_NAME);
        kpg.initialize(new ECGenParameterSpec("secp256r1"));
        return kpg.generateKeyPair();
    }

    /** Общий секрет из ECDH */
    public static byte[] ecdhSharedSecret(PrivateKey priv, PublicKey pub) throws Exception {
        KeyAgreement ka = KeyAgreement.getInstance("ECDH", BouncyCastleProvider.PROVIDER_NAME);
        ka.init(priv);
        ka.doPhase(pub, true);
        return ka.generateSecret();
    }

    /** SHA-256 хеш */
    public static byte[] sha256(byte[] data) {
        SHA256Digest digest = new SHA256Digest();
        digest.update(data, 0, data.length);
        byte[] out = new byte[digest.getDigestSize()];
        digest.doFinal(out, 0);
        return out;
    }
}
