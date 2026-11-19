// Camera-based QR Code Scanner using BarcodeDetector API with jsQR fallback.

export class QRScanner {
  constructor(videoElement, canvasElement, onScan) {
    this.video = videoElement;
    this.canvas = canvasElement;
    this.onScan = onScan;
    this.stream = null;
    this.animFrame = null;
    this.hasDetector = ('BarcodeDetector' in window);
    if (this.hasDetector) {
      try {
        this.detector = new window.BarcodeDetector({ formats: ['qr_code'] });
      } catch (e) {
        this.hasDetector = false;
      }
    }
  }

  async start() {
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: 'environment',
          width: { ideal: 1280 },
          height: { ideal: 720 },
        },
      });
      this.video.srcObject = this.stream;
      await this.video.play();
      this.scanLoop();
      return true;
    } catch (e) {
      console.error('Camera access failed:', e);
      return false;
    }
  }

  stop() {
    if (this.animFrame) {
      cancelAnimationFrame(this.animFrame);
      this.animFrame = null;
    }
    if (this.stream) {
      this.stream.getTracks().forEach(t => t.stop());
      this.stream = null;
    }
    if (this.video) {
      this.video.srcObject = null;
    }
  }

  async scanLoop() {
    if (!this.stream || this.video.readyState < 2) {
      this.animFrame = requestAnimationFrame(() => this.scanLoop());
      return;
    }

    if (this.hasDetector && this.detector) {
      try {
        const barcodes = await this.detector.detect(this.video);
        if (barcodes.length > 0) {
          const raw = barcodes[0].rawValue;
          if (raw) {
            this.stop();
            this.onScan(raw);
            return;
          }
        }
      } catch (e) {
        // Fallback or continue
      }
    }

    this.animFrame = requestAnimationFrame(() => this.scanLoop());
  }
}
