function homePage(token) {
	window.location.href = `/homePage?token=${token}`;
}
var formElement = document.querySelector('.quiz-form');
var submit = formElement.getAttribute('data-submit');
var user_id = formElement.getAttribute('data-user-id');
document.addEventListener('DOMContentLoaded', hide);
function hide() {
	if ( submit == "true" ){
		const quizForm = document.getElementById('footballQuiz');
		quizForm.style.display = 'none';
		const quizContainer = document.querySelector('.quiz-container');
		const message = document.createElement('p');
		message.textContent = 'The form has already been submitted.';
		quizContainer.appendChild(message);
	}
}
document.getElementById('footballQuiz').addEventListener('submit', function(event) {
	event.preventDefault();
	const questionCount = 5;
	let correctAnswersCount = 0;
	const questions = document.querySelectorAll(".questions")
	questions.forEach(question => {
		const selectedAnswer = question.querySelector('input[type="radio"]:checked');
		const correctAnswer = question.querySelector('#question').getAttribute('data-answer');
		if (selectedAnswer && selectedAnswer.value === correctAnswer) {
				correctAnswersCount++;
		}
	})
	const resultsContainer = document.getElementById('results');
	resultsContainer.innerHTML = `<p>You've answered ${correctAnswersCount} out of ${questionCount} questions correctly.</p>`;
	var divElement = document.querySelector('.quiz-container');
	var token = divElement.getAttribute('data-token');
	fetch(`/giveCard?user_id=${user_id}&answers=${correctAnswersCount}&token=${token}`, {
			method: 'POST'
	}).then(response => {
			if (response.ok) {
					return response.json();
			} else {
					throw new Error('Network response was not ok.');
			}
	}).then(data => {
			if(data.error) {
				alert(data.error);
			}
			else if (data.success){
				alert(data.success);
				window.location.href = `/homePage?token=${token}`;
			}
	}).catch(error => {
			console.error('There was a problem with the fetch operation:', error);
	});
});
