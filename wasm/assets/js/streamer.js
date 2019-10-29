'use strict'

/* ===========================================================================
   Libs
   =========================================================================== */
var FileReadStream = require('filestream/read')
var zlib = require('zlib')
var peek = require('peek-stream')
var through = require('through2')

/* ===========================================================================
   HTML ID tags
   =========================================================================== */
const $spinner = document.getElementById('spinner')
const $inputChecker = document.getElementById('inputChecker')
const $fastqUploader = document.getElementById('fastqUploader')
const $filedrag = document.getElementById('filedrag')
const $fastqFileName = document.getElementById('fastqFileName')
const $fastqSelecter = document.getElementById('fastqSelecter')
const $noFile = document.getElementById('noFile')

const $progressBar = document.querySelector('#progress-bar')

/* ===========================================================================
   FASTQ handling
   =========================================================================== */
// initInputFiles gets the listeners ready for FASTQ files
function initInputFiles() {
    $fastqUploader.addEventListener('change', FileSelectHandler, false)
    var xhr = new XMLHttpRequest()
    if (xhr.upload) {
        // file drop
        $filedrag.addEventListener('dragover', FileDragHover, false)
        $filedrag.addEventListener('dragleave', FileDragHover, false)
        $filedrag.addEventListener('drop', FileSelectHandler, false)
        $filedrag.style.display = 'block'
    }
}

// FileDragHover is used to cancel event and hover styling
function FileDragHover(e) {
    e.stopPropagation()
    e.preventDefault()
    e.target.className = e.type == 'dragover' ? 'hover' : ''
}

// FileSelectHandler processes any files that are added
function FileSelectHandler(e) {
    FileDragHover(e)
    resetProgress()

    // fetch FileList object
    var files = e.target.files || e.dataTransfer.files
    if (files.length != 0) {
        $fastqSelecter.classList.add('active')
        $fastqFileName.textContent = 'selected file(s):'
        $noFile.textContent = ''
    } else {
        $fastqSelecter.classList.remove('active')
        $noFile.textContent = 'no files :('
    }

    // process all the FASTQ files
    var fastqList = []
    for (var i = 0, f;
        (f = files[i]); i++) {
        // add filename to the selector bar
        var fileName = f.name.replace('C:\\fakepath\\', '')
        $noFile.innerHTML += fileName + ','

        // add the file for GROOT
        // fastqList.push(URL.createObjectURL(f));
        fastqList.push(f)
    }

    // pass the FASTQ file list to WASM
    getFiles(fastqList)
}

/* ===========================================================================
   FASTQ parsing (from: https://blog.luizirber.org/static/sourmash-wasm/app.js)
   =========================================================================== */
let fileSize = 0
let fileName = ''
let loadedFile = 0

const resetProgress = () => {
    $progressBar.style.transform = 'translateX(-100%)'
    $progressBar.style.display = 'none'
}

function isGzip(data) {
    return data[0] === 31 && data[1] === 139
}

function GzipParser() {
    return peek(function(data, swap) {
        if (isGzip(data)) return swap(null, new zlib.Unzip())
        else return swap(null, through())
    })
}

// this is the exported function - fastqStreamer - it is called by WASM when GROOT is ready to processes FASTQ data
module.exports = function(fileArr) {
    var files = fileArr[0]
    for (var i = 0; i < files.length; i++) {
        var file = files[i]
        var reader = new FileReadStream(file)
        fileSize = file.size
        fileName = file.name
        console.log('loading: ', fileName)
        $progressBar.style.display = 'block'

        reader.reader.onprogress = data => {
            loadedFile += data.loaded
            let percent = 100 - (loadedFile / fileSize) * 100
            $progressBar.style.transform = `translateX(${-percent}%)`
        }
    }

    var compressedparser = new GzipParser()
    compressedparser
        .on('data', function(data) {
            // munchFASTQ is linked to WASM and used to send the data to Go
            munchFASTQ(data, data.length)
        })
        .on('end', function(data) {
            // closeFASTQchan is a close down signal, sent to WASM once all the FASTQ data has been sent
            closeFASTQchan()
        })

    reader.pipe(compressedparser)
}

/* ===========================================================================
   GROOT set up (graph and index loading)
   =========================================================================== */
// getGraphs gets the groot graphs ready for loading
function getGraphs(graphURL) {
    var reader = new FileReader()
    fetch(graphURL)
        .then(function(response) {
            if (!response.ok) {
                statusUpdate('status', 'could not download groot graphs!')
            }
            return response.blob()
        })
        .then(data => {
            reader.readAsArrayBuffer(data)
        })
        .catch(function(error) {
            console.log(error)
        })
    reader.onload = ev => {
        var raw_data = new Uint8Array(
            ev.target.result,
            0,
            ev.target.result.byteLength
        )
        if (raw_data === null) {
            statusUpdate('status', 'could not download groot graphs!')
        } else {
            loadGraphs(graphURL, raw_data, reader.result.byteLength)
        }
    }
}

// getLSHforest gets the index ready for loading
function getLSHforest(lshfURL) {
    var reader = new FileReader()
    fetch(lshfURL)
        .then(function(response) {
            if (!response.ok) {
                statusUpdate('status', 'could not download index!')
            }
            return response.blob()
        })
        .then(data => {
            reader.readAsArrayBuffer(data)
        })
        .catch(function(error) {
            console.log(error)
        })
    reader.onload = ev => {
        var raw_data = new Uint8Array(
            ev.target.result,
            0,
            ev.target.result.byteLength
        )
        if (raw_data === null) {
            statusUpdate('status', 'could not download index!')
        } else {
            loadIndex(lshfURL, raw_data, reader.result.byteLength)
        }
    }
}

/* ===========================================================================
   Boot the app
   =========================================================================== */
const startApplication = () => {
    // setup the page
    window.onload = function() {
        window.ontouchmove = function() {
            return false
        }
        window.onorientationchange = function() {
            document.body.scrollTop = 0
        }

        // launch webassembly
        if (WebAssembly) {
            // WebAssembly.instantiateStreaming is not currently available in Safari
            if (WebAssembly && !WebAssembly.instantiateStreaming) {
                // polyfill
                WebAssembly.instantiateStreaming = async(resp, importObject) => {
                    const source = await (await resp).arrayBuffer()
                    return await WebAssembly.instantiate(source, importObject)
                }
            }
            const go = new Go()
            WebAssembly.instantiateStreaming(
                fetch('baby-groot.wasm'),
                go.importObject
            ).then(result => {
                go.run(result.instance)
                initInputFiles()
                getGraphs('assets/groot-files/dummy-db/groot.gg')
                getLSHforest('assets/groot-files/dummy-db/groot.lshf')
                $spinner.setAttribute('hidden', '')
                statusUpdate('status', 'GROOT is ready!')
            })
        } else {
            $spinner.setAttribute('hidden', '')
            console.log('WebAssembly is not supported in this browser')
            statusUpdate('status', 'please get a more recent browser!')
        }
    }

    // listen out for the input checker
    $inputChecker.addEventListener('click', function() {
        toggleDiv('inputModal')
        $spinner.removeAttribute('hidden')
        setTimeout(function() {
            inputCheck()
        }, 10)
    })

    // listen out for index selection -- TODO: I don't think this isn't listening to the right selecter
    // document.getElementById("indexSelecter").addEventListener('click', function() {
    //  var selectedIndex = document.getElementById("indexSelecter").selectedOptions;
    //  getIndex(selectedIndex[0].value);
    // });
}

startApplication()