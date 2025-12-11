// 1. Generate Identity Keys (RSA-OAEP)
export async function generateIdentityKeys() {
  return window.crypto.subtle.generateKey(
    {
      name: "RSA-OAEP",
      modulusLength: 2048,
      publicExponent: new Uint8Array([1, 0, 1]),
      hash: "SHA-256",
    },
    true,
    ["encrypt", "decrypt"]
  );
}

// 2. Export Public Key (to send to server)
export async function exportPublicKey(key: CryptoKey): Promise<string> {
  const exported = await window.crypto.subtle.exportKey("spki", key);
  return btoa(String.fromCharCode(...new Uint8Array(exported)));
}

// 3. Import Public Key (received from server)
export async function importPublicKey(pem: string): Promise<CryptoKey> {
  const binaryDerString = atob(pem);
  const binaryDer = new Uint8Array(binaryDerString.length);
  for (let i = 0; i < binaryDerString.length; i++) {
    binaryDer[i] = binaryDerString.charCodeAt(i);
  }
  return window.crypto.subtle.importKey(
    "spki",
    binaryDer.buffer,
    { name: "RSA-OAEP", hash: "SHA-256" },
    true,
    ["encrypt"]
  );
}

// 4. Encrypt Message (Hybrid: AES + RSA)
export async function encryptMessage(text: string, recipientPublicKeys: Record<string, CryptoKey>) {
  // 1. Generate the ONE Session Key (AES-GCM) used for the message
  const sessionKey = await window.crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 },
    true,
    ["encrypt"]
  );

  // 2. Encrypt the text content
  const iv = window.crypto.getRandomValues(new Uint8Array(12));
  const encoder = new TextEncoder();
  const encryptedContent = await window.crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv },
    sessionKey,
    encoder.encode(text)
  );

  // 3. Export the Session Key (so we can lock it up for others)
  const rawSessionKey = await window.crypto.subtle.exportKey("raw", sessionKey);

  // 4. Encrypt the Session Key separately for EACH recipient
  const encryptedKeys: Record<string, string> = {};
  
  for (const [userId, pubKey] of Object.entries(recipientPublicKeys)) {
      const wrappedKey = await window.crypto.subtle.encrypt(
          { name: "RSA-OAEP" },
          pubKey,
          rawSessionKey
      );
      encryptedKeys[userId] = btoa(String.fromCharCode(...new Uint8Array(wrappedKey)));
  }

  // 5. Return the full JSON payload structure
  return {
    type: 'e2ee',
    ciphertext: btoa(String.fromCharCode(...new Uint8Array(encryptedContent))),
    iv: btoa(String.fromCharCode(...iv)),
    keys: encryptedKeys
  };
}

// 5. Decrypt Message
export async function decryptMessage(
  ciphertextB64: string,
  ivB64: string,
  encryptedSessionKeyB64: string,
  myPrivateKey: CryptoKey
): Promise<string> {
  try {
    // A. Decrypt the Session Key using My Private Key
    const encKeyBuffer = Uint8Array.from(atob(encryptedSessionKeyB64), c => c.charCodeAt(0));
    const rawSessionKey = await window.crypto.subtle.decrypt(
      { name: "RSA-OAEP" },
      myPrivateKey,
      encKeyBuffer
    );

    // B. Import the Session Key
    const sessionKey = await window.crypto.subtle.importKey(
      "raw",
      rawSessionKey,
      "AES-GCM",
      false,
      ["decrypt"]
    );

    // C. Decrypt the Content
    const iv = Uint8Array.from(atob(ivB64), c => c.charCodeAt(0));
    const data = Uint8Array.from(atob(ciphertextB64), c => c.charCodeAt(0));
    
    const decrypted = await window.crypto.subtle.decrypt(
      { name: "AES-GCM", iv: iv },
      sessionKey,
      data
    );

    return new TextDecoder().decode(decrypted);
  } catch (e) {
    console.error("Decryption failed", e);
    return "🔒 Decryption Failed";
  }
}