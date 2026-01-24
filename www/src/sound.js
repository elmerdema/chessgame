let ctx;
let masterGain;
const buffers = new Map();
const rawBuffers = new Map();
let unlockPromise = null;
let unlockResolve = null;
const pendingPlays = [];
let defaultSoundsLoaded = false;

export async function initAudio() {
    if (ctx) return;
    const AudioContextCtor = window.AudioContext || window.webkitAudioContext;
    if (!AudioContextCtor) {
        throw new Error("Web Audio API is not supported in this browser.");
    }

    ctx = new AudioContextCtor();
    masterGain = ctx.createGain();
    masterGain.gain.value = 0.9;
    masterGain.connect(ctx.destination);
}

export async function loadSound(name, url) {
    const response = await fetch(url);
    const arrayBuffer = await response.arrayBuffer();
    rawBuffers.set(name, arrayBuffer);

    if (ctx && ctx.state !== "suspended") {
        const buffer = await ctx.decodeAudioData(arrayBuffer.slice(0));
        buffers.set(name, buffer);
    }
}

export async function loadAllSound(map_files) {
    const entries = map_files instanceof Map ? Array.from(map_files.entries()) : Object.entries(map_files);
    await Promise.all(entries.map(([name, url]) => loadSound(name, url)));
}

function setupUnlockListeners() {
    if (unlockPromise) return;

    unlockPromise = new Promise((resolve) => {
        unlockResolve = resolve;
    });

    const handler = async () => {
        try {
            await initAudio();
            if (ctx && ctx.state === "suspended") {
                await ctx.resume();
            }

            const decodeJobs = [];
            for (const [name, ab] of rawBuffers.entries()) {
                if (!buffers.has(name)) {
                    decodeJobs.push(
                        ctx.decodeAudioData(ab.slice(0)).then((buffer) => {
                            buffers.set(name, buffer);
                        })
                    );
                }
            }
            await Promise.all(decodeJobs);

            while (pendingPlays.length > 0) {
                const req = pendingPlays.shift();
                playNow(req.name, req.options);
            }
        } finally {
            if (typeof unlockResolve === "function") unlockResolve();
        }
    };

    const opts = { capture: true, once: true, passive: true };
    window.addEventListener("pointerdown", handler, opts);
    window.addEventListener("touchstart", handler, opts);
    window.addEventListener("keydown", handler, opts);
}

function playNow(name, options) {
    const volume = options?.volume ?? 1.0;
    const rate = options?.rate ?? 1.0;

    if (!ctx || !masterGain) return;
    const buf = buffers.get(name);
    if (!buf) return;

    const src = ctx.createBufferSource();
    src.buffer = buf;

    src.playbackRate.value = rate;
    const gain = ctx.createGain();
    gain.gain.value = volume;
    src.connect(gain);
    gain.connect(masterGain);
    src.start(0);

    return src;
}

export function play(name, options) {
    setupUnlockListeners();
    if (!ctx || ctx.state === "suspended") {
        pendingPlays.push({ name, options });
        return;
    }

    return playNow(name, options);
}

export async function loadAllSounds() {
    if (defaultSoundsLoaded) return;
    const sounds = new Map();
    sounds.set("start", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/game-start.mp3"); //done
    sounds.set("move", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/move-self.mp3");  //done
    sounds.set("capture", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/capture.mp3");//done
    sounds.set("check", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/move-check.mp3"); //done
    sounds.set("end", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/game-end.mp3"); //done
    sounds.set("promote", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/promote.mp3"); //done
    sounds.set("time", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/tenseconds.mp3");
    sounds.set("notification", "https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/notification.mp3"); //done

    await loadAllSound(sounds);
    defaultSoundsLoaded = true;
}