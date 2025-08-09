class LLMTransformers {
    constructor() {
        this.wasmLoaded = false;
        this.currentTransformation = null;
        this.transformDebounceTimer = null;
        this.formatDebounceTimer = null;
        this.stats = {
            totalTransformations: 0,
            successfulTransformations: 0,
            transformationTimes: []
        };

        this.initializeApp();
    }

    async initializeApp() {
        try {
            await this.loadWasm();
            this.wasmLoaded = true;
        } catch (error) {
            console.error('Failed to initialize WASM:', error);
            this.wasmLoaded = false;
            this.showToast('WASM failed to load, using fallback mode', 'warning');
        }

        // Always setup event listeners and load UI components
        this.setupEventListeners();
        this.loadAvailableTransformations();
        this.updateStats();
        
        // Restore state from URL parameters
        this.restoreStateFromUrl();
    }

    async loadWasm() {
        return new Promise((resolve, reject) => {
            const go = new Go();
            // Handle WASM ready message
            const messageHandler = (event) => {
                if (event.data && event.data.type === 'wasmReady') {
                    console.log('WASM ready message received:', event.data);
                    window.removeEventListener('message', messageHandler);

                    // Wait a bit for functions to be available
                    setTimeout(() => {
                        console.log('Available WASM functions:', {
                            getSupportedProviders: typeof getSupportedProviders,
                            getAvailableTransformations: typeof getAvailableTransformations,
                            transformRequest: typeof transformRequest,
                            transformResponse: typeof transformResponse,
                            transformStream: typeof transformStream,
                            transformChunk: typeof transformChunk,
                        });
                        resolve();
                    }, 100);
                }
            };

            window.addEventListener('message', messageHandler);

            console.log('Loading WASM module...');
            WebAssembly.instantiateStreaming(fetch('llm-transformers.wasm'), go.importObject)
                .then((result) => {
                    console.log('WASM instantiated, running Go program...');
                    go.run(result.instance);
                })
                .catch((error) => {
                    console.error('WASM loading error:', error);
                    window.removeEventListener('message', messageHandler);
                    reject(error);
                });

            // Timeout after 3 seconds
            setTimeout(() => {
                window.removeEventListener('message', messageHandler);
                reject(new Error('WASM loading timeout'));
            }, 3000);
        });
    }

    async reloadWasm() {
        console.log('Attempting to reload WASM module...');

        try {
            await this.loadWasm();
            this.wasmLoaded = true;
            this.showToast('WASM module reloaded successfully!', 'success');

            // Reload transformations
            this.loadAvailableTransformations();
        } catch (error) {
            console.error('Failed to reload WASM:', error);
            this.wasmLoaded = false;
            this.showToast('WASM reload failed, using fallback mode', 'warning');

            // Load fallbacks
            this.loadAvailableTransformations();
        }
    }

    setupEventListeners() {
        // Transformation type
        document.querySelectorAll('input[name="transformationType"]').forEach(radio => {
            radio.addEventListener('change', this.handleTypeChange.bind(this));
        });

        // Input controls
        const inputEditor = document.getElementById('inputEditor');
        const loadExample = document.getElementById('loadExample');
        const validateInput = document.getElementById('validateInput');

        if (inputEditor) inputEditor.addEventListener('input', this.handleInputChange.bind(this));
        if (loadExample) loadExample.addEventListener('click', this.loadExample.bind(this));
        if (validateInput) validateInput.addEventListener('click', this.validateInput.bind(this));

        // Output controls
        const copyOutput = document.getElementById('copyOutput');
        const downloadOutput = document.getElementById('downloadOutput');
        const clearOutput = document.getElementById('clearOutput');

        if (copyOutput) copyOutput.addEventListener('click', this.copyOutput.bind(this));
        if (downloadOutput) downloadOutput.addEventListener('click', this.downloadOutput.bind(this));

        // Example cards
        document.querySelectorAll('[data-example]').forEach(button => {
            button.addEventListener('click', (e) => {
                this.loadExampleByType(e.target.dataset.example);
            });
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', this.handleKeyboardShortcuts.bind(this));
    }

    handleKeyboardShortcuts(event) {
        // Ctrl/Cmd + Enter to transform
        if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
            event.preventDefault();
            this.performTransformation();
        }

        // Ctrl/Cmd + K to clear
        if ((event.ctrlKey || event.metaKey) && event.key === 'k') {
            event.preventDefault();
            this.clearInput();
            this.clearOutput();
        }
    }

    loadProviders() {
        if (!this.wasmLoaded) {
            console.log('WASM not loaded, using fallback providers');
            // Fallback providers if WASM isn't working
            this.loadFallbackProviders();
            return;
        }

        try {
            // Check if the function exists
            if (typeof getSupportedProviders !== 'function') {
                console.error('getSupportedProviders is not available');
                this.showToast('WASM functions not available, using fallback', 'warning');
                this.loadFallbackProviders();
                return;
            }

            const result = getSupportedProviders();
            console.log('getSupportedProviders result:', result);

            if (result && result.success) {
                const sourceSelect = document.getElementById('sourceProvider');
                const targetSelect = document.getElementById('targetProvider');

                sourceSelect.innerHTML = '<option value="">Select source...</option>';
                targetSelect.innerHTML = '<option value="">Select target...</option>';

                result.providers.forEach(provider => {
                    const option1 = new Option(this.formatProviderName(provider), provider);
                    const option2 = new Option(this.formatProviderName(provider), provider);
                    sourceSelect.add(option1);
                    targetSelect.add(option2);
                });

                console.log('Loaded providers:', result.providers);
            } else {
                console.error('Failed to get providers:', result ? result.error : 'No result');
                this.showToast('Failed to load providers: ' + (result ? result.error || 'Unknown error' : 'No response'), 'error');
                this.loadFallbackProviders();
            }
        } catch (error) {
            console.error('Failed to load providers:', error);
            this.showToast('Failed to load providers: ' + error.message, 'error');
            this.loadFallbackProviders();
        }
    }

    loadFallbackProviders() {
        console.log('Loading fallback providers');
        const providers = ['openai', 'gemini', 'claude'];

        const sourceSelect = document.getElementById('sourceProvider');
        const targetSelect = document.getElementById('targetProvider');

        sourceSelect.innerHTML = '<option value="">Select source...</option>';
        targetSelect.innerHTML = '<option value="">Select target...</option>';

        providers.forEach(provider => {
            const option1 = new Option(this.formatProviderName(provider), provider);
            const option2 = new Option(this.formatProviderName(provider), provider);
            sourceSelect.add(option1);
            targetSelect.add(option2);
        });

        console.log('Loaded fallback providers:', providers);
    }

    loadAvailableTransformations() {
        if (!this.wasmLoaded) {
            this.loadFallbackTransformations();
            return;
        }

        try {
            if (typeof getAvailableTransformations !== 'function') {
                console.error('getAvailableTransformations is not available');
                this.loadFallbackTransformations();
                return;
            }

            const result = getAvailableTransformations();
            if (result && result.success) {
                const container = document.getElementById('transformationsList');
                container.innerHTML = '';

                result.transformations.forEach(transformation => {
                    const card = document.createElement('div');
                    card.className = 'transformation-card';
                    card.innerHTML = `
                        <div style="font-weight: 500;">
                            ${this.formatProviderName(transformation.source)}
                        </div>
                        <div style="color: var(--text-muted); font-size: 1rem; margin: 0 0.5rem;">→</div>
                        <div style="font-weight: 500;">
                            ${this.formatProviderName(transformation.target)}
                        </div>
                    `;

                    card.addEventListener('click', () => {
                        this.selectTransformation(transformation.source, transformation.target);
                    });

                    container.appendChild(card);
                });
            } else {
                this.loadFallbackTransformations();
            }
        } catch (error) {
            console.error('Failed to load transformations:', error);
            this.loadFallbackTransformations();
        }
    }

    loadFallbackTransformations() {
        console.log('Loading fallback transformations');
        const providers = ['openai', 'gemini', 'claude'];
        const transformations = [];

        // Generate all possible transformation pairs
        for (const source of providers) {
            for (const target of providers) {
                if (source !== target) {
                    transformations.push({ source, target });
                }
            }
        }

        const container = document.getElementById('transformationsList');
        container.innerHTML = '';

        transformations.forEach(transformation => {
            const card = document.createElement('div');
            card.className = 'transformation-card';
            card.innerHTML = `
                <div style="font-weight: 500;">
                    ${this.formatProviderName(transformation.source)}
                </div>
                <div style="color: var(--text-muted); font-size: 1rem; margin: 0 0.5rem;">→</div>
                <div style="font-weight: 500;">
                    ${this.formatProviderName(transformation.target)}
                </div>
            `;

            card.addEventListener('click', () => {
                this.selectTransformation(transformation.source, transformation.target);
            });

            container.appendChild(card);
        });
    }

    selectTransformation(source, target) {
        // Update active card
        document.querySelectorAll('.transformation-card').forEach(card => {
            card.classList.remove('active');
        });
        event.target.closest('.transformation-card').classList.add('active');

        // Update provider labels in input/output sections
        document.getElementById('sourceProviderLabel').textContent =
            this.formatProviderName(source);
        document.getElementById('targetProviderLabel').textContent =
            this.formatProviderName(target);


        // Clear validation status when provider changes
        document.getElementById('inputValidation').textContent = '';
        document.getElementById('inputValidation').className = 'validation-status';

        // Store current selection for transformation
        this.currentTransformation = { source, target };

        // Update URL with new selection
        this.updateUrlState();

        // Auto-transform if there's input content
        const input = document.getElementById('inputEditor').value.trim();
        if (input) {
            this.debounceTransform();
        }
    }

    handleProviderChange() {
        const sourceProvider = document.getElementById('sourceProvider').value;
        const targetProvider = document.getElementById('targetProvider').value;

        document.getElementById('sourceProviderLabel').textContent =
            sourceProvider ? this.formatProviderName(sourceProvider) : 'Source';
        document.getElementById('targetProviderLabel').textContent =
            targetProvider ? this.formatProviderName(targetProvider) : 'Target';


        // Clear validation status when provider changes
        document.getElementById('inputValidation').textContent = '';
        document.getElementById('inputValidation').className = 'validation-status';
    }

    swapProviders() {
        const sourceSelect = document.getElementById('sourceProvider');
        const targetSelect = document.getElementById('targetProvider');

        const tempValue = sourceSelect.value;
        sourceSelect.value = targetSelect.value;
        targetSelect.value = tempValue;

        this.handleProviderChange();
    }

    handleTypeChange() {
        const type = document.querySelector('input[name="transformationType"]:checked').value;
        // Update UI based on transformation type
        console.log('Transformation type changed to:', type);
        
        // Update URL with new type selection
        this.updateUrlState();
    }

    handleInputChange() {
        const inputEditor = document.getElementById('inputEditor');
        const input = inputEditor.value;
        document.getElementById('inputCharCount').textContent = `${input.length} characters`;

        // Clear validation status when input changes
        document.getElementById('inputValidation').textContent = '';
        document.getElementById('inputValidation').className = 'validation-status';

        // Auto-adjust height based on content
        this.adjustEditorHeight(inputEditor);

        // Update URL with new input content
        this.updateUrlState();

        // Auto-transform if content is present and transformation is selected
        if (input.trim() && this.currentTransformation) {
            this.debounceTransform();
        }

        // Auto-format JSON with debounce
        this.debounceJsonFormat();
    }

    async validateInput() {
        if (!this.currentTransformation) {
            this.showToast('Please select a transformation first', 'warning');
            return;
        }

        const sourceProvider = this.currentTransformation.source;
        const input = document.getElementById('inputEditor').value.trim();
        const validationEl = document.getElementById('inputValidation');

        if (!input) {
            validationEl.textContent = 'Empty input';
            validationEl.className = 'validation-status invalid';
            return;
        }

        try {
            const result = validateRequest(sourceProvider, input);
            if (result.success && result.isValid) {
                validationEl.textContent = '✓ Valid';
                validationEl.className = 'validation-status valid';
            } else {
                validationEl.textContent = '✗ Invalid: ' + (result.error || 'Unknown error');
                validationEl.className = 'validation-status invalid';
            }
        } catch (error) {
            validationEl.textContent = '✗ Validation error: ' + error.message;
            validationEl.className = 'validation-status invalid';
        }
    }

    async loadExample() {
        if (!this.currentTransformation) {
            this.showToast('Please select a transformation first', 'warning');
            return;
        }

        const sourceProvider = this.currentTransformation.source;

        try {
            const result = getExampleRequest(sourceProvider);
            if (result.success) {
                document.getElementById('inputEditor').value = result.example;
                this.handleInputChange();
                this.showToast('Example loaded successfully', 'success');
                // Auto-transform the loaded example
                if (this.currentTransformation) {
                    this.debounceTransform();
                }
            } else {
                this.showToast('Failed to load example: ' + result.error, 'error');
            }
        } catch (error) {
            this.showToast('Failed to load example: ' + error.message, 'error');
        }
    }

    loadExampleByType(exampleType) {
        if (!this.currentTransformation) {
            this.showToast('Please select a transformation first', 'warning');
            return;
        }

        const sourceProvider = this.currentTransformation.source;
        const transformationType = document.querySelector('input[name="transformationType"]:checked').value;

        const examples = this.getExamples();

        // First check if the example type exists
        if (!examples[exampleType]) {
            this.showToast(`Unknown example type: ${exampleType}`, 'error');
            return;
        }

        // Check if transformation type exists for this example
        if (!examples[exampleType][transformationType]) {
            this.showToast(`${exampleType} examples not available for ${transformationType} transformations`, 'warning');
            return;
        }

        // Check if provider exists for this example+transformation combination
        const example = examples[exampleType][transformationType][sourceProvider];
        if (!example) {
            const availableProviders = Object.keys(examples[exampleType][transformationType]);
            this.showToast(`${exampleType} example not available for ${sourceProvider} (available: ${availableProviders.join(', ')})`, 'warning');
            return;
        }

        // Load the example
        document.getElementById('inputEditor').value = JSON.stringify(example, null, 2);
        this.handleInputChange();
        this.showToast('Example loaded successfully', 'success');

        // Auto-transform the loaded example
        if (this.currentTransformation) {
            this.debounceTransform();
        }
    }

    getExamples() {
        return {
            'simple-chat': {
                request: {
                    openai: {
                        model: "gpt-4",
                        messages: [
                            { role: "system", content: "You are a helpful assistant." },
                            { role: "user", content: "Hello! How can you help me today?" }
                        ],
                        max_tokens: 150,
                        temperature: 0.7
                    },
                    gemini: {
                        contents: [
                            {
                                role: "user",
                                parts: [{ text: "Hello! How can you help me today?" }]
                            }
                        ],
                        systemInstruction: {
                            parts: [{ text: "You are a helpful assistant." }]
                        },
                        generationConfig: {
                            maxOutputTokens: 150,
                            temperature: 0.7
                        }
                    },
                    claude: {
                        model: "claude-3-5-sonnet-20241022",
                        max_tokens: 150,
                        temperature: 0.7,
                        system: "You are a helpful assistant.",
                        messages: [
                            { role: "user", content: "Hello! How can you help me today?" }
                        ]
                    }
                },
                response: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion",
                        created: 1677652288,
                        model: "gpt-4",
                        choices: [
                            {
                                index: 0,
                                message: {
                                    role: "assistant",
                                    content: "Hello! I'm here to help you with a wide variety of tasks. I can assist with writing, analysis, math, coding, creative projects, answering questions, and much more. What would you like to work on today?"
                                },
                                finish_reason: "stop"
                            }
                        ],
                        usage: { prompt_tokens: 12, completion_tokens: 35, total_tokens: 47 }
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        { text: "Hello! I'm here to help you with a wide variety of tasks. I can assist with writing, analysis, math, coding, creative projects, answering questions, and much more. What would you like to work on today?" }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP",
                                index: 0
                            }
                        ],
                        usageMetadata: { promptTokenCount: 12, candidatesTokenCount: 35, totalTokenCount: 47 }
                    },
                    claude: {
                        id: "msg_01EhYXf6n7Dp5gXxXkFp9mGV",
                        type: "message",
                        role: "assistant",
                        content: [
                            { type: "text", text: "Hello! I'm here to help you with a wide variety of tasks. I can assist with writing, analysis, math, coding, creative projects, answering questions, and much more. What would you like to work on today?" }
                        ],
                        model: "claude-3-5-sonnet-20241022",
                        stop_reason: "end_turn",
                        usage: { input_tokens: 12, output_tokens: 35 }
                    }
                }
            },
            'function-calls': {
                request: {
                    openai: {
                        model: "gpt-4",
                        messages: [
                            { role: "user", content: "What's the weather like in New York?" }
                        ],
                        tools: [
                            {
                                type: "function",
                                function: {
                                    name: "get_weather",
                                    description: "Get current weather information for a location",
                                    parameters: {
                                        type: "object",
                                        properties: {
                                            location: { type: "string", description: "The location to get weather for" }
                                        },
                                        required: ["location"]
                                    }
                                }
                            }
                        ]
                    },
                    gemini: {
                        contents: [
                            {
                                role: "user",
                                parts: [{ text: "Search for the latest AI research papers" }]
                            }
                        ],
                        tools: [
                            { googleSearch: {} }
                        ]
                    },
                    claude: {
                        model: "claude-3-5-sonnet-20241022",
                        max_tokens: 1024,
                        messages: [
                            { role: "user", content: "List the files in the current directory" }
                        ],
                        tools: [
                            {
                                name: "bash",
                                description: "Run bash commands",
                                input_schema: {
                                    type: "object",
                                    properties: {
                                        command: { type: "string", description: "The bash command to run" }
                                    },
                                    required: ["command"]
                                }
                            }
                        ]
                    }
                },
                response: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion",
                        created: 1677652288,
                        model: "gpt-4",
                        choices: [
                            {
                                index: 0,
                                message: {
                                    role: "assistant",
                                    content: null,
                                    tool_calls: [
                                        {
                                            id: "call_abc123",
                                            type: "function",
                                            function: {
                                                name: "get_weather",
                                                arguments: '{"location": "New York"}'
                                            }
                                        }
                                    ]
                                },
                                finish_reason: "tool_calls"
                            }
                        ],
                        usage: { prompt_tokens: 45, completion_tokens: 12, total_tokens: 57 }
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        {
                                            functionCall: {
                                                name: "googleSearch",
                                                args: { query: "latest AI research papers 2024" }
                                            }
                                        }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP",
                                index: 0
                            }
                        ],
                        usageMetadata: { promptTokenCount: 15, candidatesTokenCount: 8, totalTokenCount: 23 }
                    },
                    claude: {
                        id: "msg_01EhYXf6n7Dp5gXxXkFp9mGV",
                        type: "message",
                        role: "assistant",
                        content: [
                            {
                                type: "tool_use",
                                id: "toolu_01A09q90qw90lq",
                                name: "bash",
                                input: { command: "ls -la" }
                            }
                        ],
                        model: "claude-3-5-sonnet-20241022",
                        stop_reason: "tool_use",
                        usage: { input_tokens: 25, output_tokens: 18 }
                    }
                },
                stream: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion.chunk",
                        created: 1677652288,
                        model: "gpt-4",
                        choices: [
                            {
                                index: 0,
                                delta: {
                                    tool_calls: [
                                        {
                                            id: "call_abc123",
                                            function: {
                                                name: "get_weather",
                                                arguments: '{"location": "New York"}'
                                            }
                                        }
                                    ]
                                },
                                finish_reason: "tool_calls"
                            }
                        ]
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        {
                                            functionCall: {
                                                name: "googleSearch",
                                                args: { query: "latest AI research papers 2024" }
                                            }
                                        }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP"
                            }
                        ]
                    },
                    claude: {
                        type: "content_block_delta",
                        index: 0,
                        delta: {
                            type: "tool_use",
                            id: "toolu_01A09q90qw90lq",
                            name: "bash",
                            input: { command: "ls -la" }
                        }
                    }
                },
                chunk: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion.chunk",
                        created: 1677652288,
                        model: "gpt-4",
                        choices: [
                            {
                                index: 0,
                                delta: {
                                    tool_calls: [
                                        {
                                            index: 0,
                                            id: "call_abc123",
                                            function: { name: "get_weather" }
                                        }
                                    ]
                                },
                                finish_reason: null
                            }
                        ]
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [{ functionCall: { name: "googleSearch" } }],
                                    role: "model"
                                },
                                index: 0
                            }
                        ]
                    },
                    claude: {
                        type: "content_block_start",
                        index: 0,
                        content_block: {
                            type: "tool_use",
                            id: "toolu_01A09q90qw90lq",
                            name: "bash"
                        }
                    }
                }
            },
            'vision': {
                request: {
                    openai: {
                        model: "gpt-4-vision-preview",
                        messages: [
                            {
                                role: "user",
                                content: [
                                    { type: "text", text: "What do you see in this image?" },
                                    {
                                        type: "image_url",
                                        image_url: {
                                            url: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
                                        }
                                    }
                                ]
                            }
                        ],
                        max_tokens: 300
                    },
                    gemini: {
                        contents: [
                            {
                                role: "user",
                                parts: [
                                    { text: "What do you see in this image?" },
                                    {
                                        inlineData: {
                                            mimeType: "image/jpeg",
                                            data: "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
                                        }
                                    }
                                ]
                            }
                        ],
                        generationConfig: { maxOutputTokens: 300 }
                    },
                    claude: {
                        model: "claude-3-5-sonnet-20241022",
                        max_tokens: 300,
                        messages: [
                            {
                                role: "user",
                                content: [
                                    { type: "text", text: "What do you see in this image?" },
                                    {
                                        type: "image",
                                        source: {
                                            type: "base64",
                                            media_type: "image/jpeg",
                                            data: "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="
                                        }
                                    }
                                ]
                            }
                        ]
                    }
                },
                response: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion",
                        created: 1677652288,
                        model: "gpt-4-vision-preview",
                        choices: [
                            {
                                index: 0,
                                message: {
                                    role: "assistant",
                                    content: "I can see a very small image that appears to be mostly white or blank. The image seems to be minimal with very little visual content to describe."
                                },
                                finish_reason: "stop"
                            }
                        ],
                        usage: { prompt_tokens: 85, completion_tokens: 25, total_tokens: 110 }
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        { text: "I can see a very small image that appears to be mostly white or blank. The image seems to be minimal with very little visual content to describe." }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP",
                                index: 0
                            }
                        ],
                        usageMetadata: { promptTokenCount: 85, candidatesTokenCount: 25, totalTokenCount: 110 }
                    },
                    claude: {
                        id: "msg_01EhYXf6n7Dp5gXxXkFp9mGV",
                        type: "message",
                        role: "assistant",
                        content: [
                            { type: "text", text: "I can see a very small image that appears to be mostly white or blank. The image seems to be minimal with very little visual content to describe." }
                        ],
                        model: "claude-3-5-sonnet-20241022",
                        stop_reason: "end_turn",
                        usage: { input_tokens: 85, output_tokens: 25 }
                    }
                },
                stream: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion.chunk",
                        created: 1677652288,
                        model: "gpt-4-vision-preview",
                        choices: [
                            {
                                index: 0,
                                delta: {
                                    content: "I can see a very small image that appears to be mostly white or blank."
                                },
                                finish_reason: "stop"
                            }
                        ]
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        { text: "I can see a very small image that appears to be mostly white or blank." }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP"
                            }
                        ]
                    },
                    claude: {
                        type: "message_delta",
                        delta: {
                            type: "text_delta",
                            text: "I can see a very small image that appears to be mostly white or blank."
                        }
                    }
                },
                chunk: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion.chunk",
                        created: 1677652288,
                        model: "gpt-4-vision-preview",
                        choices: [
                            {
                                index: 0,
                                delta: { content: "I can see" },
                                finish_reason: null
                            }
                        ]
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [{ text: "I can see" }],
                                    role: "model"
                                },
                                index: 0
                            }
                        ]
                    },
                    claude: {
                        type: "content_block_delta",
                        index: 0,
                        delta: {
                            type: "text_delta",
                            text: "I can see"
                        }
                    }
                }
            },
            'provider-special': {
                request: {
                    gemini: {
                        contents: [
                            {
                                role: "user",
                                parts: [{ text: "Write and execute Python code to calculate fibonacci numbers" }]
                            }
                        ],
                        tools: [
                            { codeExecution: {} }
                        ]
                    },
                    claude: {
                        model: "claude-3-5-sonnet-20241022",
                        max_tokens: 1024,
                        system: "You have access to bash and computer use tools.",
                        messages: [
                            { role: "user", content: "Check system memory usage and create a simple Python script" }
                        ],
                        tools: [
                            {
                                name: "bash",
                                description: "Run bash commands",
                                input_schema: {
                                    type: "object",
                                    properties: {
                                        command: { type: "string" }
                                    },
                                    required: ["command"]
                                }
                            },
                            {
                                name: "computer_20241022",
                                description: "Use computer to take screenshots and interact with desktop",
                                input_schema: {
                                    type: "object",
                                    properties: {
                                        action: { type: "string", enum: ["screenshot", "click", "type"] }
                                    },
                                    required: ["action"]
                                }
                            }
                        ]
                    },
                    openai: {
                        model: "gpt-4",
                        messages: [
                            { role: "user", content: "Create a simple calculator function" }
                        ],
                        tools: [
                            {
                                type: "function",
                                function: {
                                    name: "python_executor",
                                    description: "Execute Python code",
                                    parameters: {
                                        type: "object",
                                        properties: {
                                            code: { type: "string", description: "Python code to execute" }
                                        },
                                        required: ["code"]
                                    }
                                }
                            }
                        ]
                    }
                },
                response: {
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        { text: "I'll write Python code to calculate Fibonacci numbers:" },
                                        {
                                            executableCode: {
                                                language: "python",
                                                code: "def fibonacci(n):\n    if n <= 1:\n        return n\n    return fibonacci(n-1) + fibonacci(n-2)\n\nprint([fibonacci(i) for i in range(10)])"
                                            }
                                        },
                                        {
                                            codeExecutionResult: {
                                                outcome: "OUTCOME_OK",
                                                output: "[0, 1, 1, 2, 3, 5, 8, 13, 21, 34]"
                                            }
                                        }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP",
                                index: 0
                            }
                        ],
                        usageMetadata: { promptTokenCount: 20, candidatesTokenCount: 45, totalTokenCount: 65 }
                    },
                    claude: {
                        id: "msg_01EhYXf6n7Dp5gXxXkFp9mGV",
                        type: "message",
                        role: "assistant",
                        content: [
                            { type: "text", text: "I'll check system memory and create a Python script for you." },
                            {
                                type: "tool_use",
                                id: "toolu_01A09q90qw90lq",
                                name: "bash",
                                input: { command: "free -h" }
                            }
                        ],
                        model: "claude-3-5-sonnet-20241022",
                        stop_reason: "tool_use",
                        usage: { input_tokens: 35, output_tokens: 28 }
                    }
                },
                stream: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion.chunk",
                        created: 1677652288,
                        model: "gpt-4",
                        choices: [
                            {
                                index: 0,
                                delta: {
                                    role: "assistant",
                                    content: "Hello! I'm here to help you with a wide variety of tasks."
                                },
                                finish_reason: "stop"
                            }
                        ]
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [
                                        { text: "Hello! I'm here to help you with a wide variety of tasks." }
                                    ],
                                    role: "model"
                                },
                                finishReason: "STOP",
                                index: 0
                            }
                        ],
                        usageMetadata: { promptTokenCount: 12, candidatesTokenCount: 15, totalTokenCount: 27 }
                    },
                    claude: {
                        type: "message_delta",
                        delta: {
                            type: "text_delta",
                            text: "Hello! I'm here to help you with a wide variety of tasks."
                        },
                        usage: { output_tokens: 15 }
                    }
                },
                chunk: {
                    openai: {
                        id: "chatcmpl-8pQ0e0Z0Y1z7X5d9G7z7p8pQ",
                        object: "chat.completion.chunk",
                        created: 1677652288,
                        model: "gpt-4",
                        choices: [
                            {
                                index: 0,
                                delta: { content: "Hello" },
                                finish_reason: null
                            }
                        ]
                    },
                    gemini: {
                        candidates: [
                            {
                                content: {
                                    parts: [{ text: "Hello" }],
                                    role: "model"
                                },
                                index: 0
                            }
                        ]
                    },
                    claude: {
                        type: "content_block_delta",
                        index: 0,
                        delta: {
                            type: "text_delta",
                            text: "Hello"
                        }
                    }
                }
            }
        }
    }

    clearInput() {
        document.getElementById('inputEditor').value = '';
        document.getElementById('inputCharCount').textContent = '0 characters';
        document.getElementById('inputValidation').textContent = '';
        document.getElementById('inputValidation').className = 'validation-status';
    }

    debounceJsonFormat() {
        // Clear existing timer
        if (this.formatDebounceTimer) {
            clearTimeout(this.formatDebounceTimer);
        }

        // Set new timer for 2 second delay (ideal interval for auto-formatting)
        this.formatDebounceTimer = setTimeout(() => {
            this.autoFormatJson();
        }, 2000);
    }

    autoFormatJson() {
        const inputEditor = document.getElementById('inputEditor');
        const input = inputEditor.value.trim();
        
        if (!input) return;

        try {
            // Try to parse and format the JSON
            const parsed = JSON.parse(input);
            const formatted = JSON.stringify(parsed, null, 2);
            
            // Only update if the formatting actually changed
            if (formatted !== input) {
                const cursorPos = inputEditor.selectionStart;
                inputEditor.value = formatted;
                
                // Restore cursor position approximately
                const newPos = Math.min(cursorPos, formatted.length);
                inputEditor.setSelectionRange(newPos, newPos);
                
                // Adjust height after formatting
                this.adjustEditorHeight(inputEditor);
            }
        } catch (error) {
            // Invalid JSON, don't format
            console.log('Invalid JSON, skipping auto-format');
        }
    }

    adjustEditorHeight(editor) {
        // Reset height to auto to get the correct scrollHeight
        editor.style.height = 'auto';

        // Calculate the required height based on content
        const scrollHeight = editor.scrollHeight;
        const minHeight = 200; // min-height from CSS
        const maxHeight = window.innerHeight * 0.8; // 80vh

        // Set the height to fit content, respecting min/max limits
        const newHeight = Math.max(minHeight, Math.min(scrollHeight + 2, maxHeight));
        editor.style.height = newHeight + 'px';

        // Also adjust output display height to match
        const outputDisplay = document.getElementById('outputDisplay');
        if (outputDisplay) {
            outputDisplay.style.height = newHeight + 'px';
        }
    }

    debounceTransform() {
        // Clear existing timer
        if (this.transformDebounceTimer) {
            clearTimeout(this.transformDebounceTimer);
        }

        // Set new timer for 1 second delay
        this.transformDebounceTimer = setTimeout(() => {
            this.performTransformation();
        }, 1000);
    }

    async performTransformation() {
        if (!this.currentTransformation) {
            this.showToast('Please select a transformation first', 'warning');
            return;
        }

        const sourceProvider = this.currentTransformation.source;
        const targetProvider = this.currentTransformation.target;
        const input = document.getElementById('inputEditor').value.trim();
        const transformationType = document.querySelector('input[name="transformationType"]:checked').value;

        if (!input) {
            this.clearOutput();
            return;
        }

        const startTime = performance.now();

        try {
            let result;

            // Check if WASM is available and functions exist
            if (!this.wasmLoaded) {
                throw new Error('WASM transformation functions are not available. Please reload the page.');
            }

            // Use the appropriate function based on transformation type
            switch (transformationType) {
                case 'request':
                    if (typeof transformRequest !== 'function') {
                        throw new Error('transformRequest function is not available');
                    }
                    result = transformRequest(sourceProvider, targetProvider, input);
                    break;
                case 'response':
                    if (typeof transformResponse !== 'function') {
                        throw new Error('transformResponse function is not available');
                    }
                    result = transformResponse(sourceProvider, targetProvider, input);
                    break;
                case 'stream':
                    if (typeof transformStream !== 'function') {
                        throw new Error('transformStream function is not available');
                    }
                    result = transformStream(sourceProvider, targetProvider, input);
                    break;
                case 'chunk':
                    if (typeof transformChunk !== 'function') {
                        throw new Error('transformChunk function is not available');
                    }
                    result = transformChunk(sourceProvider, targetProvider, input);
                    break;
                default:
                    throw new Error('Unknown transformation type: ' + transformationType);
            }

            const endTime = performance.now();
            const transformTime = Math.round(endTime - startTime);

            if (result && result.success) {
                this.displayOutput(result.result, transformTime);
                this.showToast('Transformation completed successfully!', 'success');

                // Update stats
                this.stats.totalTransformations++;
                this.stats.successfulTransformations++;
                this.stats.transformationTimes.push(transformTime);
                this.updateStats();
            } else {
                throw new Error(result ? result.error || 'Unknown transformation error' : 'No result returned');
            }
        } catch (error) {
            console.error('Transformation error:', error);

            // Check if it's the "Go program has already exited" error
            if (error.message.includes('Go program has already exited')) {
                this.showToast('WASM module crashed. Attempting to reload...', 'warning');
                this.showStatus('WASM module crashed - attempting reload', 'loading');
                this.wasmLoaded = false;

                // Attempt auto-reload once
                setTimeout(() => {
                    this.reloadWasm();
                }, 1000);
            } else {
                this.showToast('Transformation failed: ' + error.message, 'error');
            }

            // Update stats
            this.stats.totalTransformations++;
            this.updateStats();

            // Show error in output
            this.displayError(error.message);
        } finally {
            // Cleanup is automatically handled since no button UI to reset
        }
    }

    displayOutput(jsonString, transformTime) {
        const outputEl = document.getElementById('outputDisplay');
        const outputCharCount = document.getElementById('outputCharCount');
        const transformationTime = document.getElementById('transformationTime');

        // Format JSON with syntax highlighting
        try {
            const parsed = JSON.parse(jsonString);
            const formatted = JSON.stringify(parsed, null, 2);

            outputEl.innerHTML = `<pre><code class="language-json">${this.escapeHtml(formatted)}</code></pre>`;

            // Apply syntax highlighting if Prism is available
            if (window.Prism) {
                Prism.highlightAllUnder(outputEl);
            }

            outputCharCount.textContent = `${formatted.length} characters`;
            transformationTime.textContent = `Transformed in ${transformTime}ms`;

            // Auto-adjust output height based on content
            this.adjustOutputHeight(outputEl);

        } catch (error) {
            // If JSON parsing fails, display as plain text
            outputEl.textContent = jsonString;
            outputCharCount.textContent = `${jsonString.length} characters`;
            transformationTime.textContent = `Transformed in ${transformTime}ms`;

            // Auto-adjust output height for plain text
            this.adjustOutputHeight(outputEl);
        }
    }

    adjustOutputHeight(outputEl) {
        // Calculate the required height based on content
        const scrollHeight = outputEl.scrollHeight;
        const minHeight = 200; // min-height from CSS
        const maxHeight = window.innerHeight * 0.8; // 80vh

        // Set the height to fit content, respecting min/max limits
        const newHeight = Math.max(minHeight, Math.min(scrollHeight + 20, maxHeight));
        outputEl.style.height = newHeight + 'px';
    }

    displayError(errorMessage) {
        const outputEl = document.getElementById('outputDisplay');
        outputEl.innerHTML = `<div style="color: var(--error-color); padding: 1rem; text-align: center;">
            <strong>Transformation Error</strong><br>
            ${this.escapeHtml(errorMessage)}
        </div>`;

        document.getElementById('outputCharCount').textContent = '0 characters';
        document.getElementById('transformationTime').textContent = '';

        // Auto-adjust height for error display
        this.adjustOutputHeight(outputEl);
    }

    copyOutput() {
        const outputEl = document.getElementById('outputDisplay');
        const text = outputEl.textContent;

        if (!text || text.includes('Transformed JSON will appear here')) {
            this.showToast('Nothing to copy', 'warning');
            return;
        }

        navigator.clipboard.writeText(text).then(() => {
            this.showToast('Output copied to clipboard!', 'success');
        }).catch(() => {
            this.showToast('Failed to copy to clipboard', 'error');
        });
    }

    downloadOutput() {
        const outputEl = document.getElementById('outputDisplay');
        const text = outputEl.textContent;

        if (!text || text.includes('Transformed JSON will appear here')) {
            this.showToast('Nothing to download', 'warning');
            return;
        }

        const sourceProvider = document.getElementById('sourceProvider').value;
        const targetProvider = document.getElementById('targetProvider').value;
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const filename = `${sourceProvider}-to-${targetProvider}-${timestamp}.json`;

        const blob = new Blob([text], { type: 'application/json' });
        const url = URL.createObjectURL(blob);

        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        this.showToast('Output downloaded!', 'success');
    }

    clearOutput() {
        const outputEl = document.getElementById('outputDisplay');
        outputEl.innerHTML = '<div class="placeholder">Transformed JSON will appear here...</div>';
        document.getElementById('outputCharCount').textContent = '0 characters';
        document.getElementById('transformationTime').textContent = '';
    }

    updateStats() {
        const totalEl = document.getElementById('totalTransformations');
        const successRateEl = document.getElementById('successRate');
        const avgTimeEl = document.getElementById('avgTime');

        totalEl.textContent = this.stats.totalTransformations;

        const successRate = this.stats.totalTransformations > 0
            ? Math.round((this.stats.successfulTransformations / this.stats.totalTransformations) * 100)
            : 100;
        successRateEl.textContent = `${successRate}%`;

        const avgTime = this.stats.transformationTimes.length > 0
            ? Math.round(this.stats.transformationTimes.reduce((a, b) => a + b, 0) / this.stats.transformationTimes.length)
            : 0;
        avgTimeEl.textContent = `${avgTime}ms`;
    }

    formatProviderName(provider) {
        const names = {
            openai: 'OpenAI',
            gemini: 'Gemini',
            claude: 'Claude'
        };
        return names[provider] || provider;
    }

    showStatus(message, type) {
        const statusEl = document.getElementById('wasmStatus');
        const reloadBtn = document.getElementById('reloadWasm');

        statusEl.textContent = message;
        statusEl.className = `status ${type}`;

        // Show reload button for error states
        if (type === 'error' || message.includes('crashed')) {
            reloadBtn.style.display = 'inline-block';
        } else {
            reloadBtn.style.display = 'none';
        }

        // Update WASM info in footer
        const wasmInfo = document.getElementById('wasmInfo');
        if (type === 'ready') {
            wasmInfo.textContent = 'WASM module loaded • Go runtime active';
        } else if (type === 'error') {
            wasmInfo.textContent = 'WASM module error • Using fallback mode';
        }
    }

    showToast(message, type) {
        const toast = document.getElementById('toast');
        toast.textContent = message;
        toast.className = `toast ${type}`;

        // Show toast
        setTimeout(() => toast.classList.add('show'), 10);

        // Hide toast after 4 seconds
        setTimeout(() => {
            toast.classList.remove('show');
        }, 4000);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // URL State Management
    updateUrlState() {
        const url = new URL(window.location);
        const params = url.searchParams;

        // Store transformation type
        const transformationType = document.querySelector('input[name="transformationType"]:checked')?.value;
        if (transformationType) {
            params.set('type', transformationType);
        }

        // Store provider selection
        if (this.currentTransformation) {
            params.set('source', this.currentTransformation.source);
            params.set('target', this.currentTransformation.target);
        }

        // Store input content (compressed using base64)
        const inputContent = document.getElementById('inputEditor').value.trim();
        if (inputContent) {
            try {
                // Compress JSON by removing whitespace if it's valid JSON
                const compressed = JSON.stringify(JSON.parse(inputContent));
                const encoded = btoa(encodeURIComponent(compressed));
                params.set('input', encoded);
            } catch (error) {
                // If not valid JSON, store as-is (base64 encoded)
                const encoded = btoa(encodeURIComponent(inputContent));
                params.set('input', encoded);
            }
        } else {
            params.delete('input');
        }

        // Update URL without page refresh
        window.history.replaceState({}, '', url);
    }

    restoreStateFromUrl() {
        const url = new URL(window.location);
        const params = url.searchParams;

        // Restore transformation type
        const transformationType = params.get('type');
        if (transformationType) {
            const typeRadio = document.querySelector(`input[name="transformationType"][value="${transformationType}"]`);
            if (typeRadio) {
                typeRadio.checked = true;
            }
        }

        // Restore input content
        const inputParam = params.get('input');
        if (inputParam) {
            try {
                const decoded = decodeURIComponent(atob(inputParam));
                document.getElementById('inputEditor').value = decoded;

                // Trigger input change handler to update character count and adjust height
                const inputEditor = document.getElementById('inputEditor');
                this.adjustEditorHeight(inputEditor);
                document.getElementById('inputCharCount').textContent = `${decoded.length} characters`;

                // Auto-format the restored JSON content
                this.autoFormatJson();
            } catch (error) {
                console.error('Failed to restore input from URL:', error);
            }
        }

        // Restore provider selection
        const source = params.get('source');
        const target = params.get('target');
        if (source && target) {
            // Set the transformation after a short delay to ensure DOM is ready
            setTimeout(() => {
                this.selectTransformationByParams(source, target);
            }, 100);
        }
    }

    selectTransformationByParams(source, target) {
        // Find and activate the appropriate transformation card
        const cards = document.querySelectorAll('.transformation-card');
        cards.forEach(card => {
            card.classList.remove('active');
        });

        // Update provider labels
        document.getElementById('sourceProviderLabel').textContent = this.formatProviderName(source);
        document.getElementById('targetProviderLabel').textContent = this.formatProviderName(target);

        // Store current selection
        this.currentTransformation = { source, target };

        // Find and activate the matching card
        cards.forEach(card => {
            // Get the card's innerHTML to check the structure
            const cardHtml = card.innerHTML;
            
            // Extract the source and target from the card structure
            // Cards have structure: <div>Source</div><div>→</div><div>Target</div>
            const divs = card.querySelectorAll('div');
            if (divs.length >= 3) {
                const cardSource = divs[0].textContent.trim();
                const cardTarget = divs[2].textContent.trim();
                
                if (cardSource === this.formatProviderName(source) && 
                    cardTarget === this.formatProviderName(target)) {
                    card.classList.add('active');
                }
            }
        });

        // Auto-transform if there's input content
        const input = document.getElementById('inputEditor').value.trim();
        if (input) {
            this.debounceTransform();
        }
    }
}

// Initialize the application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new LLMTransformers();
});