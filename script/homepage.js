function collection(token) {
	window.location.href = `/collection?token=${token}`;
}
function market(token) {
	window.location.href = `/market?token=${token}`;
}
function quiz(token) {
	window.location.href = `/dailyQuestions?token=${token}`;
}

function chat(token) {
	window.location.href = `/chatHandler?token=${token}`;
}