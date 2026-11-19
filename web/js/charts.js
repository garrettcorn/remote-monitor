// Canvas-based real-time sparkline charts for CPU, RAM, and Network history.

export class SparklineChart {
  constructor(canvasId, options = {}) {
    this.canvas = document.getElementById(canvasId);
    if (!this.canvas) return;
    this.ctx = this.canvas.getContext('2d');
    this.options = {
      maxPoints: options.maxPoints || 60,
      color: options.color || '#38bdf8',
      fillColor: options.fillColor || 'rgba(56, 189, 248, 0.15)',
      min: options.min !== undefined ? options.min : 0,
      max: options.max !== undefined ? options.max : 100,
      autoScale: options.autoScale || false,
      unit: options.unit || '%',
      ...options,
    };
    this.data = [];

    // Handle high DPI displays
    this.resize();
    window.addEventListener('resize', () => this.resize());
  }

  resize() {
    if (!this.canvas) return;
    const rect = this.canvas.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    this.canvas.width = rect.width * dpr;
    this.canvas.height = rect.height * dpr;
    this.ctx.scale(dpr, dpr);
    this.draw();
  }

  push(value) {
    if (value === undefined || value === null || isNaN(value)) return;
    this.data.push(value);
    if (this.data.length > this.options.maxPoints) {
      this.data.shift();
    }
    this.draw();
  }

  clear() {
    this.data = [];
    this.draw();
  }

  draw() {
    if (!this.canvas || !this.ctx) return;
    const rect = this.canvas.getBoundingClientRect();
    const w = rect.width;
    const h = rect.height;
    const ctx = this.ctx;

    ctx.clearRect(0, 0, w, h);

    if (this.data.length < 2) return;

    let minVal = this.options.min;
    let maxVal = this.options.max;

    if (this.options.autoScale) {
      minVal = Math.min(...this.data);
      maxVal = Math.max(...this.data);
      if (minVal === maxVal) {
        maxVal += 1;
      }
    }

    const range = (maxVal - minVal) || 1;
    const stepX = w / (this.options.maxPoints - 1);
    const startX = w - (this.data.length - 1) * stepX;

    // Draw subtle grid lines
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(0, h * 0.25);
    ctx.lineTo(w, h * 0.25);
    ctx.moveTo(0, h * 0.5);
    ctx.lineTo(w, h * 0.5);
    ctx.moveTo(0, h * 0.75);
    ctx.lineTo(w, h * 0.75);
    ctx.stroke();

    // Build curve path
    ctx.beginPath();
    const points = [];
    for (let i = 0; i < this.data.length; i++) {
      const x = startX + i * stepX;
      const normalized = (this.data[i] - minVal) / range;
      const y = h - (normalized * (h - 8) + 4);
      points.push({ x, y });
      if (i === 0) {
        ctx.moveTo(x, y);
      } else {
        // Smooth line
        const prev = points[i - 1];
        const cpX = (prev.x + x) / 2;
        ctx.bezierCurveTo(cpX, prev.y, cpX, y, x, y);
      }
    }

    // Stroke line
    ctx.strokeStyle = this.options.color;
    ctx.lineWidth = 2;
    ctx.stroke();

    // Fill area under curve
    if (points.length > 1) {
      const last = points[points.length - 1];
      const first = points[0];
      ctx.lineTo(last.x, h);
      ctx.lineTo(first.x, h);
      ctx.closePath();

      const gradient = ctx.createLinearGradient(0, 0, 0, h);
      gradient.addColorStop(0, this.options.fillColor);
      gradient.addColorStop(1, 'rgba(0, 0, 0, 0)');
      ctx.fillStyle = gradient;
      ctx.fill();
    }
  }
}
