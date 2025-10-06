import Anthropic from '@anthropic-ai/sdk';

async function main() {
  const apiKey = process.env.ANTHROPIC_API_KEY;
  if (!apiKey) {
    console.error('ANTHROPIC_API_KEY environment variable not set');
    process.exit(1);
  }

  const client = new Anthropic({
    apiKey: apiKey,
    baseURL: 'http://localhost:4567', // This is where we're configuring mirra
  });

  try {
    const message = await client.messages.create({
      model: 'claude-3-5-sonnet-20241022',
      max_tokens: 1024,
      messages: [
        {
          role: 'user',
          content: 'Say hello and a joke',
        },
      ],
    });

    if (message.content.length === 0) {
      console.log('No response from Claude.');
      return;
    }

    for (const block of message.content) {
      if (block.type === 'text') {
        console.log(block.text);
      }
    }
  } catch (error) {
    console.error('Failed to create message:', error);
    process.exit(1);
  }
}

main();
