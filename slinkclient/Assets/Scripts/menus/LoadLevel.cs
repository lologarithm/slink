using UnityEngine;
using System.Collections;

public class LoadLevel : MonoBehaviour {

	public GameObject loadingImage;

	public void LoadScene(string level)
	{
		UnityEngine.SceneManagement.SceneManager.LoadScene(level);
	}
}
