#!/usr/bin/env python3
"""
Text Detection Script for Image Validation
Uses OCR to detect any text/characters in generated images
"""

import sys
import os
from PIL import Image
import pytesseract
import re

def detect_text_in_image(image_path, confidence_threshold=30):
    """
    Detect text in an image using OCR
    
    Args:
        image_path: Path to the image file
        confidence_threshold: Minimum confidence to consider text detected (0-100)
    
    Returns:
        dict with 'has_text' (bool) and 'detected_text' (str)
    """
    try:
        if not os.path.exists(image_path):
            return {"error": f"Image not found: {image_path}"}
        
        # Open image
        img = Image.open(image_path)
        
        # Run OCR with detailed data
        data = pytesseract.image_to_data(img, output_type=pytesseract.Output.DICT)
        
        detected_texts = []
        text_count = 0
        
        # Check each detected text element
        for i, conf in enumerate(data['conf']):
            if int(conf) > confidence_threshold:
                text = data['text'][i].strip()
                # Filter out single characters and common false positives
                if len(text) > 1 and re.search(r'[a-zA-Z0-9]{2,}', text):
                    detected_texts.append(f"{text} (conf: {conf})")
                    text_count += 1
        
        has_text = text_count > 0
        
        return {
            "has_text": has_text,
            "text_count": text_count,
            "detected_text": ", ".join(detected_texts) if detected_texts else "None",
            "image_path": image_path
        }
        
    except Exception as e:
        return {"error": str(e)}

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 text_detector.py <image_path> [confidence_threshold]")
        sys.exit(1)
    
    image_path = sys.argv[1]
    confidence = int(sys.argv[2]) if len(sys.argv) > 2 else 30
    
    result = detect_text_in_image(image_path, confidence)
    
    if "error" in result:
        print(f"ERROR: {result['error']}")
        sys.exit(2)
    
    # Output result as simple format for Go to parse
    if result['has_text']:
        print(f"TEXT_DETECTED:{result['text_count']}:{result['detected_text']}")
        sys.exit(1)  # Exit code 1 = text detected
    else:
        print("NO_TEXT_DETECTED")
        sys.exit(0)  # Exit code 0 = clean image
