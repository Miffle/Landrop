console.log("Landrop frontend loaded");

// -------------------------
// STATE
// -------------------------
let ws = null;
let myId = null;
let devices = [];
let fileBuffers = {};   // incoming file state: { [fileId]: { name, size, received, chunks[] } }
let selectedFile = null;
let isSending = false;

const CHUNK_SIZE = 64 * 1024; // 64 KiB — fast on LAN, low server memory

// -------------------------
// CONNECT
// -------------------------
function connectToWebsocket() {
    const name = document.getElementById("deviceName").value.trim() || "Anonymous";
    myId = getClientId();

    const proto = location.protocol === "https:" ? "wss" : "ws";
    const url = `${proto}://${location.host}/ws`;

    hideError();
    addLog("🔄 Подключение к " + url);

    ws = new WebSocket(url);

    ws.onopen = () => {
        document.getElementById("myName").textContent = name;
        document.getElementById("connectBtn").textContent = "Подключено ✓";
        document.getElementById("connectBtn").disabled = true;
        hideError();

        ws.send(JSON.stringify({
            type: "register",
            payload: { id: myId, name }
        }));

        addLog("🟢 Подключено как «" + name + "»");
    };

    ws.onmessage = (evt) => handleMessage(JSON.parse(evt.data));

    ws.onclose = (evt) => {
        const reason = evt.reason ? ` (${evt.reason})` : "";
        addLog(`🔴 Соединение закрыто [код ${evt.code}]${reason}`);
        if (evt.code !== 1000) {
            showError(`Не удалось подключиться (код ${evt.code}). Проверьте, что сервер запущен и порт открыт в брандмауэре.`);
        }
        document.getElementById("connectBtn").textContent = "Подключиться";
        document.getElementById("connectBtn").disabled = false;
    };

    ws.onerror = () => {
        addLog("⚠️ Ошибка WebSocket — проверьте сеть и брандмауэр");
        showError("Ошибка подключения. Убедитесь, что: 1) сервер запущен, 2) порт 6437 разрешён в брандмауэре Windows.");
    };
}

// -------------------------
// MESSAGE ROUTER
// -------------------------
function handleMessage(data) {
    switch (data.type) {

        case "devices":
            devices = (data.payload.devices || []).filter(d => d.id !== myId);
            renderDevices();
            renderSelect();
            break;

        case "file_start":
            onFileStart(data.payload);
            break;

        case "file_chunk":
            onFileChunk(data.payload);
            break;

        case "file_end":
            onFileEnd(data.payload);
            break;

        case "direct_message":
            addLog("📩 От " + (data.payload.from || "?") + ": " + data.payload.text);
            break;
    }
}

// -------------------------
// INCOMING FILE HANDLING
// -------------------------
function onFileStart(p) {
    fileBuffers[p.fileId] = {
        name: p.name,
        size: p.size,
        received: 0,
        chunks: []
    };
    addLog("📨 Входящий файл: «" + p.name + "» (" + formatSize(p.size) + ")");
}

function onFileChunk(p) {
    const buf = fileBuffers[p.fileId];
    if (!buf) return;

    // decode base64 → Uint8Array
    const binary = atob(p.data);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);

    buf.chunks.push(bytes);
    buf.received += bytes.length;
}

function onFileEnd(p) {
    const buf = fileBuffers[p.fileId];
    if (!buf) return;

    const blob = new Blob(buf.chunks);
    addIncomingFile(buf.name, blob);
    delete fileBuffers[p.fileId];
    addLog("✅ Файл «" + buf.name + "» получен");
}

function addIncomingFile(name, blob) {
    const card = document.getElementById("incomingCard");
    card.style.display = "";

    const url = URL.createObjectURL(blob);

    const div = document.createElement("div");
    div.className = "incoming-file";
    div.innerHTML = `
        <span class="file-icon">📄</span>
        <span class="file-name">${escHtml(name)}</span>
        <span class="file-size">${formatSize(blob.size)}</span>
        <a href="${url}" download="${escHtml(name)}" class="btn-download">⬇ Скачать</a>
    `;
    document.getElementById("incomingList").prepend(div);
}

// -------------------------
// FILE SEND
// -------------------------
function onFileSelected(input) {
    const file = input.files[0];
    if (!file) return;
    selectedFile = file;

    document.getElementById("dropZoneText").textContent = "Файл выбран";
    document.getElementById("selectedFileName").textContent = file.name;
    document.getElementById("selectedFileSize").textContent = formatSize(file.size);
    document.getElementById("selectedFile").style.display = "flex";
}

function clearFile() {
    selectedFile = null;
    document.getElementById("fileInput").value = "";
    document.getElementById("selectedFile").style.display = "none";
    document.getElementById("dropZoneText").textContent = "📂 Перетащите файл сюда или нажмите для выбора";
    setProgress(0);
    document.getElementById("progressWrap").style.display = "none";
}

async function sendFile() {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        addLog("⚠️ Сначала подключитесь");
        return;
    }
    if (!selectedFile) {
        addLog("⚠️ Выберите файл");
        return;
    }
    const to = document.getElementById("recipient").value;
    if (!to) {
        addLog("⚠️ Выберите получателя");
        return;
    }
    if (isSending) return;

    isSending = true;
    const file = selectedFile;
    const fileId = generateUUID();
    const total = file.size;
    let offset = 0;

    // --- file_start ---
    ws.send(JSON.stringify({
        type: "file_start",
        payload: { to, fileId, name: file.name, size: total }
    }));

    document.getElementById("progressWrap").style.display = "";
    document.getElementById("sendBtn").disabled = true;
    addLog("📤 Отправка «" + file.name + "» (" + formatSize(total) + ")");

    // --- send chunks ---
    while (offset < total) {
        const slice = file.slice(offset, offset + CHUNK_SIZE);
        const buffer = await slice.arrayBuffer();

        // encode to base64
        const bytes = new Uint8Array(buffer);
        let binary = "";
        for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
        const b64 = btoa(binary);

        ws.send(JSON.stringify({
            type: "file_chunk",
            payload: { to, fileId, data: b64 }
        }));

        offset += buffer.byteLength;
        setProgress(Math.min(100, Math.round(offset / total * 100)));

        // yield to browser every 16 chunks (~4 MiB) to keep UI responsive
        if ((offset / CHUNK_SIZE) % 16 === 0) {
            await new Promise(r => setTimeout(r, 0));
        }
    }

    // --- file_end ---
    ws.send(JSON.stringify({
        type: "file_end",
        payload: { to, fileId }
    }));

    addLog("✅ Отправлено: «" + file.name + "»");
    isSending = false;
    document.getElementById("sendBtn").disabled = false;

    // reset after 2 sec
    setTimeout(clearFile, 2000);
}

function setProgress(pct) {
    document.getElementById("progressBar").style.width = pct + "%";
    document.getElementById("progressText").textContent = pct + "%";
}

// -------------------------
// DRAG AND DROP
// -------------------------
const dropZone = document.getElementById("dropZone");

["dragenter", "dragover"].forEach(ev =>
    dropZone?.addEventListener(ev, e => { e.preventDefault(); dropZone.classList.add("drag-over"); })
);
["dragleave", "drop"].forEach(ev =>
    dropZone?.addEventListener(ev, e => { e.preventDefault(); dropZone.classList.remove("drag-over"); })
);
dropZone?.addEventListener("drop", e => {
    const file = e.dataTransfer.files[0];
    if (!file) return;
    selectedFile = file;
    document.getElementById("fileInput").files; // just reference
    document.getElementById("dropZoneText").textContent = "Файл выбран";
    document.getElementById("selectedFileName").textContent = file.name;
    document.getElementById("selectedFileSize").textContent = formatSize(file.size);
    document.getElementById("selectedFile").style.display = "flex";
});

// -------------------------
// UI: DEVICES
// -------------------------
function renderDevices() {
    const list = document.getElementById("devicesList");
    list.innerHTML = "";

    if (devices.length === 0) {
        list.innerHTML = "<li class='empty-hint'>Нет устройств</li>";
        return;
    }

    devices.forEach(d => {
        const li = document.createElement("li");
        li.className = "device-item";
        li.textContent = `${d.name}`;
        const id = document.createElement("span");
        id.className = "device-id";
        id.textContent = d.id.slice(0, 8) + "…";
        li.appendChild(id);
        list.appendChild(li);
    });
}

function renderSelect() {
    const select = document.getElementById("recipient");
    const prev = select.value;
    select.innerHTML = "";

    devices.forEach(d => {
        const option = document.createElement("option");
        option.value = d.id;
        option.textContent = d.name;
        select.appendChild(option);
    });

    if (prev) select.value = prev;
}

// -------------------------
// CHAT
// -------------------------
function sendDirectMessageFromUI() {
    const toId = document.getElementById("recipient").value;
    const text = document.getElementById("message").value.trim();
    if (!ws || !toId || !text) return;

    ws.send(JSON.stringify({
        type: "direct_message",
        payload: { to: toId, text }
    }));

    addLog("➡️ " + text);
    document.getElementById("message").value = "";
}

// -------------------------
// HELPERS
// -------------------------
function showError(msg) {
    const el = document.getElementById("wsError");
    if (!el) return;
    el.textContent = msg;
    el.style.display = "block";
}

function hideError() {
    const el = document.getElementById("wsError");
    if (el) el.style.display = "none";
}

function getClientId() {
    let id = localStorage.getItem("landrop_client_id");
    if (!id) {
        id = generateUUID();
        localStorage.setItem("landrop_client_id", id);
    }
    return id;
}

// crypto.randomUUID() requires HTTPS; this works over plain HTTP too
function generateUUID() {
    if (typeof crypto !== "undefined" && crypto.getRandomValues) {
        return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, c =>
            (c ^ crypto.getRandomValues(new Uint8Array(1))[0] & 15 >> c / 4).toString(16)
        );
    }
    // last-resort fallback
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = Math.random() * 16 | 0;
        return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
    });
}

function generateName() {
    const adj = ["Fast", "Cool", "Silent", "Smart", "Red", "Blue", "Bold", "Calm"];
    const noun = ["Fox", "Laptop", "Phone", "Tiger", "Node", "Pixel", "Hawk", "Bear"];
    document.getElementById("deviceName").value =
        adj[Math.floor(Math.random() * adj.length)] + " " +
        noun[Math.floor(Math.random() * noun.length)];
}

function addLog(text) {
    const log = document.getElementById("log");
    const div = document.createElement("div");
    div.className = "log-entry";
    const ts = new Date().toLocaleTimeString();
    div.innerHTML = `<span class="log-ts">${ts}</span> ${escHtml(text)}`;
    log.prepend(div);
}

function formatSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " КБ";
    if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + " МБ";
    return (bytes / 1024 / 1024 / 1024).toFixed(2) + " ГБ";
}

function escHtml(str) {
    return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// -------------------------
// CLEANUP
// -------------------------
window.addEventListener("beforeunload", () => ws?.close(1000, "Page unload"));