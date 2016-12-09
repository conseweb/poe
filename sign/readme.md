# 签名

安全性加强： 按照唯一的Key值，生成公私钥， 签名时采用32位的字符为影响因子

# 验证

java 或 C# 验证签名示例如下： 
× 需要引用第三方包 “BouncyCastle.Crypto”

```

static void Main(string[] args)
        {
            byte[] pukRaw = Org.BouncyCastle.Utilities.Encoders.Hex.Decode(@"040161bfa897777543eace1c5cb8a550bc87fb51d3c7f601a022285f8cd88af0cc2f821d3c3edf38f67cd084f8f0d7445e6d80a285848b1657d708dc05a44bb70b76e901c558e385f2d0b7030e03f5e21acc85a899eb89159b515fc1d52e0ea922d1f1b8d640afac781031ba37dfff9c5e35a68602468a5527e8bcf64db2c7f8ab960948f7");
            byte[] signRaw = Org.BouncyCastle.Utilities.Encoders.Hex.Decode(@"3081880242014bdb9475687ff4753cb6e5e7336c8fb518c84e1dfb2c4432ad7aeeb2ce51a26f9bcfc3b418d85128749a7bde1b0139bcf5eb29632cc7cc607d0db4814ad78e42c802420123cd9069877cc222fbf5f4dda70acd2c55445bb147c96fc8b1f75942d76e888006b9efcf4eed01e4a4c78f1b7ae5041b8688d4ae973ba02ee5a3b0450934e09902");
            byte[] msgRaw = Encoding.UTF8.GetBytes(@"我叫MT");
            var signer = new Org.BouncyCastle.Crypto.Signers.ECDsaSigner();
            Org.BouncyCastle.Asn1.X9.X9ECParameters secp521r1 = Org.BouncyCastle.Asn1.Sec.SecNamedCurves.GetByName("secp521r1");
            Org.BouncyCastle.Crypto.Parameters.ECDomainParameters ecParames = new Org.BouncyCastle.Crypto.Parameters.ECDomainParameters(secp521r1.Curve, secp521r1.G, secp521r1.N, secp521r1.H);
            //Org.BouncyCastle.Crypto.Parameters.ECPublicKeyParameters pukParame = new Org.BouncyCastle.Crypto.Parameters.ECPublicKeyParameters(ecParames.Curve.DecodePoint(ConvertKeyFormat(pukRaw)),ecParames);
            Org.BouncyCastle.Crypto.Parameters.ECPublicKeyParameters pukParame = new Org.BouncyCastle.Crypto.Parameters.ECPublicKeyParameters(ecParames.Curve.DecodePoint(pukRaw), ecParames);
            signer.Init(false, pukParame);
            Org.BouncyCastle.Asn1.DerSequence seq = (Org.BouncyCastle.Asn1.DerSequence)new Org.BouncyCastle.Asn1.Asn1InputStream(signRaw).ReadObject();
            Org.BouncyCastle.Asn1.DerInteger r = (Org.BouncyCastle.Asn1.DerInteger)seq[0];
            Org.BouncyCastle.Asn1.DerInteger s = (Org.BouncyCastle.Asn1.DerInteger)seq[1];
            System.Security.Cryptography.SHA512 hash = new System.Security.Cryptography.SHA512CryptoServiceProvider();
            byte[] hashBytes = hash.ComputeHash(msgRaw);
            Console.WriteLine(signer.VerifySignature(hashBytes, r.Value, s.Value));
            Console.ReadLine();
        }

```

