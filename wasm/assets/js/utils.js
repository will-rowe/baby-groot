// a selection of JS functions for running the GROOT WASM port

// getElementById
function $id(id) {
	return document.getElementById(id);
}

// gotMem sets the webassembly linear memory with the image buffer result at the slice header pointer passed from Go
function gotMem(pointer) {
	memoryBytes.set(bytes, pointer);
}

function toggleDiv(id) {
    var el = document.getElementById(id);
    if (el.style.display === "block") {
        el.style.display = "none";
    } else {
        el.style.display = "block";
    }
}

function statusUpdate(elementID, msg) {
	var element = document.getElementById(elementID);
    $(element).stop().fadeOut(500, function() {
        element.innerHTML = msg;
        $(element, this).fadeIn(500);
    });
}

function iconUpdate(id) {
    var el = document.getElementById(id);
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-check');
        el.style.color = '#80ff00';
        $(el, this).fadeIn(500);
    });
}

function setParameters() {
    iconUpdate("paramIcon")
    statusUpdate("status", "parameters are set")
}

function startSpinner() {
    var el = document.getElementById("start");
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-compact-disc', "fa-spin");
        el.style.color = '#F16721';
        $(el, this).fadeIn(500);
    });
}
 
function stopSpinner() {
    var el = document.getElementById("start");
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-play');
        el.style.color = '#E9DBC5';
        $(el, this).fadeIn(500);
    });
}

function startLogo() {
    var svgObj = document.getElementById("logo-animated").contentDocument;
    var l1 = svgObj.getElementById("leaf-1");
    var l2 = svgObj.getElementById("leaf-2");
    var l3 = svgObj.getElementById("leaf-3");    
    l1.classList.add('growing');
    l2.classList.add('growing');
    l3.classList.add('growing');
}

function stopLogo() {
    var svgObj = document.getElementById("logo-animated").contentDocument;
    var l1 = svgObj.getElementById("leaf-1");
    var l2 = svgObj.getElementById("leaf-2");
    var l3 = svgObj.getElementById("leaf-3");    
    l1.classList.remove('growing');
    l2.classList.remove('growing');
    l3.classList.remove('growing');
}

function addResults(ref, abun) {
    $("#resultsContent").append("<tr><td>" + ref + "</td><td>" + abun + "</td></tr>");
}

function updateTimer(elapsedTime) {
    $("#runTime").append("<sub style='color:black;'>time elapsed: " + elapsedTime + "</sub>");
}

$('#uploader1').bind('change', function () {
    var filename = $("#uploader1").val();
    if (/^\s*$/.test(filename)) {
      $("#fastqSelecter").removeClass('active');
      $("#noFile").text("No file chosen..."); 
    }
    else {
      $("#fastqSelecter").addClass('active');
      $("#noFile").text(filename.replace("C:\\fakepath\\", "")); 
    }
  });
  

$('#uploader2').bind('change', function () {
    var filename = $("#uploader2").val();
    if (/^\s*$/.test(filename)) {
      $("#indexSelecter").removeClass('active');
      $("#noIndexFile").text("No file chosen..."); 
    }
    else {
      $("#indexSelecter").addClass('active');
      $("#noIndexFile").text(filename.replace("C:\\fakepath\\", "")); 
    }
  });