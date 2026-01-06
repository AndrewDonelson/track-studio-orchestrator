#!/usr/bin/env python3
"""
WhisperX API Service for CQAI
Provides REST API endpoints for audio transcription and timing extraction
"""

import os
import subprocess
import tempfile
import shutil
import json
from pathlib import Path
from typing import Optional, Dict, Any
import uvicorn
from fastapi import FastAPI, UploadFile, File, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import logging
# Import torch only when needed
# import torch
# Import whisperx only when needed to avoid startup issues
WHISPERX_AVAILABLE = True

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="WhisperX API", description="Audio transcription and timing API using WhisperX")

class TranscriptionRequest(BaseModel):
    audio_url: Optional[str] = None
    language: str = "en"
    model: str = "large-v2"
    align_mode: bool = False
    lyrics_text: Optional[str] = None

class TranscriptionResponse(BaseModel):
    status: str
    job_id: str
    message: str

class TranscriptionStatus(BaseModel):
    status: str
    result: Optional[Dict[str, Any]] = None
    error: Optional[str] = None

# In-memory job storage (in production, use Redis or database)
jobs = {}

def run_whisperx_script(audio_path: str, language: str = "en", model: str = "large-v2",
                       align_mode: bool = False, lyrics_file: Optional[str] = None) -> Dict[str, Any]:
    """Run WhisperX directly and return results"""

    try:
        import torch
        import whisper
    except ImportError as e:
        return {"error": f"Failed to import required packages: {e}"}

    logger.info(f"Processing audio file: {audio_path} with language: {language}, model: {model}")

    try:
        # Use GPU 1 (RTX 3090)
        device = "cuda:1"
        logger.info(f"Using device: {device}")
        
        # Check CUDA availability
        if not torch.cuda.is_available():
            return {"error": "CUDA not available"}
        
        if torch.cuda.device_count() < 2:
            return {"error": f"GPU 1 not available, only {torch.cuda.device_count()} GPUs found"}
        
        # Use basic whisper model
        logger.info("Loading Whisper model...")
        model = whisper.load_model(model, device=device)
        logger.info("Model loaded successfully")
        
        # Transcribe
        logger.info("Starting transcription...")
        result = model.transcribe(audio_path, language=language)
        logger.info("Transcription completed")
        
        # Convert to desired format
        transcription = result["text"]
        srt_content = ""
        vtt_content = "WEBVTT\n\n"
        json_data = result
        
        segment_id = 1
        for segment in result["segments"]:
            start_time = segment["start"]
            end_time = segment["end"]
            text = segment["text"]
            
            # SRT format
            srt_content += f"{segment_id}\n"
            srt_content += f"{format_timestamp(start_time)} --> {format_timestamp(end_time)}\n"
            srt_content += f"{text}\n\n"
            
            # VTT format
            vtt_content += f"{format_timestamp_vtt(start_time)} --> {format_timestamp_vtt(end_time)}\n"
            vtt_content += f"{text}\n\n"
            
            segment_id += 1
        
        return {
            "transcription": transcription.strip(),
            "srt": srt_content.strip(),
            "vtt": vtt_content.strip(),
            "json_data": json_data
        }

    except Exception as e:
        logger.error(f"Error running Whisper: {str(e)}")
        return {"error": str(e)}

def format_timestamp(seconds: float) -> str:
    """Format seconds to SRT timestamp format (HH:MM:SS,mmm)"""
    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)
    secs = int(seconds % 60)
    millis = int((seconds % 1) * 1000)
    return f"{hours:02d}:{minutes:02d}:{secs:02d},{millis:03d}"

def format_timestamp_vtt(seconds: float) -> str:
    """Format seconds to VTT timestamp format (HH:MM:SS.mmm)"""
    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)
    secs = int(seconds % 60)
    millis = int((seconds % 1) * 1000)
    return f"{hours:02d}:{minutes:02d}:{secs:02d}.{millis:03d}"

def process_transcription(job_id: str, audio_path: str, language: str, model: str,
                        align_mode: bool, lyrics_file: Optional[str]):
    """Background task to process transcription"""
    try:
        result = run_whisperx_script(audio_path, language, model, align_mode, lyrics_file)
        jobs[job_id] = {"status": "completed", "result": result}
    except Exception as e:
        jobs[job_id] = {"status": "failed", "error": str(e)}
    finally:
        # Cleanup temporary files
        try:
            os.unlink(audio_path)
            if lyrics_file:
                os.unlink(lyrics_file)
        except:
            pass

@app.post("/transcribe", response_model=TranscriptionResponse)
async def transcribe_audio(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    language: str = "en",
    model: str = "large-v2",
    align_mode: bool = False,
    lyrics: Optional[str] = None
):
    """Upload audio file and start transcription"""

    # Validate file type
    if not file.filename.lower().endswith(('.wav', '.mp3', '.m4a', '.flac', '.ogg')):
        raise HTTPException(status_code=400, detail="Unsupported file type. Use WAV, MP3, M4A, FLAC, or OGG")

    # Save uploaded file temporarily
    with tempfile.NamedTemporaryFile(delete=False, suffix=Path(file.filename).suffix) as temp_file:
        shutil.copyfileobj(file.file, temp_file)
        audio_path = temp_file.name

    # Save lyrics if provided
    lyrics_path = None
    if lyrics and align_mode:
        with tempfile.NamedTemporaryFile(delete=False, suffix='.txt', mode='w') as lyrics_file:
            lyrics_file.write(lyrics)
            lyrics_path = lyrics_file.name

    # Generate job ID
    job_id = f"job_{len(jobs) + 1}"

    # Start background processing
    jobs[job_id] = {"status": "processing"}
    background_tasks.add_task(
        process_transcription,
        job_id, audio_path, language, model, align_mode, lyrics_path
    )

    return TranscriptionResponse(
        status="accepted",
        job_id=job_id,
        message="Transcription started. Check status with GET /status/{job_id}"
    )

@app.get("/status/{job_id}", response_model=TranscriptionStatus)
async def get_transcription_status(job_id: str):
    """Get transcription job status"""
    if job_id not in jobs:
        raise HTTPException(status_code=404, detail="Job not found")

    job = jobs[job_id]
    return TranscriptionStatus(**job)

@app.post("/transcribe/sync")
async def transcribe_audio_sync(
    file: UploadFile = File(...),
    language: str = "en",
    model: str = "large-v2",
    align_mode: bool = False,
    lyrics: Optional[str] = None
):
    """Synchronous transcription (not recommended for large files)"""

    # Validate file type
    if not file.filename.lower().endswith(('.wav', '.mp3', '.m4a', '.flac', '.ogg')):
        raise HTTPException(status_code=400, detail="Unsupported file type")

    # Save uploaded file temporarily
    with tempfile.NamedTemporaryFile(delete=False, suffix=Path(file.filename).suffix) as temp_file:
        shutil.copyfileobj(file.file, temp_file)
        audio_path = temp_file.name

    # Save lyrics if provided
    lyrics_path = None
    if lyrics and align_mode:
        with tempfile.NamedTemporaryFile(delete=False, suffix='.txt', mode='w') as lyrics_file:
            lyrics_file.write(lyrics)
            lyrics_path = lyrics_file.name

    try:
        result = run_whisperx_script(audio_path, language, model, align_mode, lyrics_path)

        if "error" in result:
            raise HTTPException(status_code=500, detail=result["error"])

        return result

    finally:
        # Cleanup
        try:
            os.unlink(audio_path)
            if lyrics_path:
                os.unlink(lyrics_path)
        except:
            pass

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "WhisperX API"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8181)