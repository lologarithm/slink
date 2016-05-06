using UnityEngine;
using UnityEngine.UI;
using System.Collections;

public class LoadLevel : MonoBehaviour {

	public GameObject loadingImage;

	public void LoadScene(string level)
	{
		UnityEngine.SceneManagement.SceneManager.LoadScene(level);
	}

    public void SetName() {
        GameObject textbox = GameObject.Find("usernameText");
        Text t = textbox.GetComponent<Text>();
        if (t.text == null || t.text == "") {
            // Random name?
            t.text = "asdf";
        }
        Debug.Log("Setting name to: " + t.text);
        PlayerPrefs.SetString("name", t.text);
    }
}
