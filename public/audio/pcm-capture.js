// Capture on the audio rendering thread; transfer 20 ms PCM frames without copies.
class PCMCapture extends AudioWorkletProcessor {
  constructor(options) {
    super();
    this.targetRate = options.processorOptions.sampleRate;
    this.ratio = sampleRate / this.targetRate;
    this.position = 0;
    this.sum = 0;
    this.count = 0;
    this.frame = new Int16Array(Math.round(this.targetRate * 0.02));
    this.offset = 0;
  }
  process(inputs) {
    const input = inputs[0]?.[0];
    if (!input) return true;
    for (let i = 0; i < input.length; i++) {
      this.sum += input[i];
      this.count++;
      this.position++;
      if (this.position >= this.ratio) {
        const sample = Math.max(-1, Math.min(1, this.sum / this.count));
        this.frame[this.offset++] = sample < 0 ? sample * 32768 : sample * 32767;
        this.position -= this.ratio;
        this.sum = 0;
        this.count = 0;
        if (this.offset === this.frame.length) {
          this.port.postMessage(this.frame.buffer, [this.frame.buffer]);
          this.frame = new Int16Array(Math.round(this.targetRate * 0.02));
          this.offset = 0;
        }
      }
    }
    return true;
  }
}
registerProcessor('pcm-capture', PCMCapture);
