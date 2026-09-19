// Renders a ZBP transfer payload as a QR image. The payload bytes are UTF-8, so Polish letters survive bank app scanning.

import qrcode from "qrcode-generator";

export function TransferQr({ payload }: { payload: string }) {
  const code = qrcode(0, "M");
  // The library reads one byte per character, so the UTF-8 bytes enter as a byte string.
  code.addData(String.fromCharCode(...new TextEncoder().encode(payload)), "Byte");
  code.make();
  return <figure className="transfer-qr"><img src={code.createDataURL(5, 2)} alt="Kod QR przelewu do zeskanowania w aplikacji banku" width={code.getModuleCount() * 5 + 20} /><figcaption>Zeskanuj w aplikacji banku, w opcji płatności kodem QR.</figcaption></figure>;
}
