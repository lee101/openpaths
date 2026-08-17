export interface ImageDemo {
  imageUrl?: string;
  outputUrl: string;
  prompt?: string;
  payload: Record<string, unknown>;
  title: string;
  description: string;
}

export const IMAGE_DEMOS: Record<string, ImageDemo> = {
  'flux-2-pro-preview': {
    title: 'FLUX.2 Pro Preview generation',
    description: 'A live BFL API generation using the default prompt upsampler, fixed safety tolerance 5, and a web-ready WebP output.',
    outputUrl: '/static/image-gallery/bfl/flux-2-pro-preview-routing-orchid.webp',
    prompt: 'A precision glass greenhouse at night containing a miniature Black Forest, luminous orchid petals tracing elegant routing paths through moss and ferns, cinematic product photography, subtle blue and amber light, physically accurate glass reflections, no text, no logos',
    payload: {
      model: 'flux-2-pro-preview',
      prompt: 'A precision glass greenhouse at night containing a miniature Black Forest, luminous orchid petals tracing elegant routing paths through moss and ferns, cinematic product photography, subtle blue and amber light, physically accurate glass reflections, no text, no logos',
      size: '1024x1024',
      n: 1,
      output_format: 'webp',
      safety_tolerance: 5,
      disable_pup: false,
    },
  },
  'fal-ai/hidream-o1-image/edit': {
    title: 'Prompted image edit example',
    description: 'Use a reference image plus an edit prompt to replace a product while preserving the photographed scene.',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/lipstick.png',
    prompt: 'Replace the perfume bottle with a lipstick',
    payload: {
      model: 'fal-ai/hidream-o1-image/edit',
      prompt: 'Replace the perfume bottle with a lipstick',
      reference_image_urls: [
        'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg',
      ],
      image_size: 'landscape_16_9',
      num_inference_steps: 50,
      guidance_scale: 5,
      num_images: 1,
      output_format: 'png',
      enable_safety_checker: false,
    },
  },
  'fal-ai/flux-2-pro/outpaint': {
    title: 'Outpainted image example',
    description: 'Expand the canvas to the left, right, and bottom while keeping the original scene coherent.',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/flux-outpaint/input.png',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/flux-outpaint/output.jpg',
    payload: {
      model: 'fal-ai/flux-2-pro/outpaint',
      image_url: 'https://openpathsstatic.openpaths.io/static/uploads/playground/flux-outpaint/input.png',
      expand_top: 0,
      expand_bottom: 200,
      expand_left: 200,
      expand_right: 200,
      enable_safety_checker: true,
      output_format: 'jpeg',
    },
  },
};
