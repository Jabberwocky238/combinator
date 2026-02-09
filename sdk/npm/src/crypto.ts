/**
 * HMAC signature utilities
 * Equivalent to Go's hmac package with SHA-256
 */
export class HMACSignature {
  /**
   * Generate HMAC-SHA256 signature
   * Equivalent to Go's GenerateHMACSignature function
   * @param secretKey - The secret key for HMAC
   * @param data - The data to sign (string or Uint8Array)
   * @returns Base64URL encoded signature (without padding)
   */
  static async generate(secretKey: string, data: string | Uint8Array): Promise<string> {
    const encoder = new TextEncoder()
    const keyData = encoder.encode(secretKey)
    const messageData = typeof data === 'string' ? encoder.encode(data) : data

    const cryptoKey = await crypto.subtle.importKey(
      'raw',
      keyData,
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign']
    )

    const signature = await crypto.subtle.sign('HMAC', cryptoKey, messageData as BufferSource)

    // Convert to base64url (without padding) - same as Go's base64.RawURLEncoding
    return this.arrayBufferToBase64Url(signature)
  }

  /**
   * Verify HMAC-SHA256 signature
   * Equivalent to Go's VerifyHMACSignature function
   * @param secretKey - The secret key for HMAC
   * @param data - The data to verify
   * @param signature - The signature to verify against
   * @throws Error with message "invalid signature" if verification fails
   */
  static async verify(secretKey: string, data: string | Uint8Array, signature: string): Promise<void> {
    const expected = await this.generate(secretKey, data)
    if (!this.constantTimeEqual(expected, signature)) {
      throw new Error('invalid signature')
    }
  }

  /**
   * Convert ArrayBuffer to Base64URL string (without padding)
   * Equivalent to Go's base64.RawURLEncoding.EncodeToString
   */
  private static arrayBufferToBase64Url(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer)
    let binary = ''
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i])
    }
    return btoa(binary)
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=/g, '')
  }

  /**
   * Constant-time string comparison
   * Equivalent to Go's hmac.Equal to prevent timing attacks
   */
  private static constantTimeEqual(a: string, b: string): boolean {
    if (a.length !== b.length) {
      return false
    }
    let result = 0
    for (let i = 0; i < a.length; i++) {
      result |= a.charCodeAt(i) ^ b.charCodeAt(i)
    }
    return result === 0
  }
}
